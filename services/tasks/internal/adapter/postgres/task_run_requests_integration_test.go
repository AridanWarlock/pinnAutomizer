package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/fileservice"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	tasksAfterRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/afterRun"
	tasksDelete "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/delete"
	tasksRun "github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks/run"
)

func TestTaskIntegrationRunRequestSurvivesDeliveryAndTaskDeletion(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	ctx, task := integrationTask(t, repo, domain.TaskStatusCreated)
	input := tasksRun.Input{TaskID: task.ID}
	require.NoError(t, tasksRun.New(repo).RunTask(ctx, input))
	// Outbox delivery is not the source of truth for accepted HTTP requests.
	_, err := db.Exec(ctx, "DELETE FROM events")
	require.NoError(t, err)
	require.NoError(t, tasksRun.New(repo).RunTask(ctx, input))
	stored, err := repo.GetTaskByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusInQueue, stored.Status)
	files := fileservice.NewFileService()
	require.NoError(t, tasksAfterRun.New(repo, files).UpdateTaskAfterRun(ctx, tasksAfterRun.Input{ID: task.ID}))
	require.NoError(t, tasksDelete.New(repo, files).DeleteTask(ctx, tasksDelete.Input{TaskID: task.ID}))
	require.NoError(t, tasksRun.New(repo).RunTask(ctx, input))
	var events, requests int
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&events))
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM task_run_requests").Scan(&requests))
	assert.Zero(t, events)
	assert.Equal(t, 1, requests)
}

func TestTaskIntegrationRunRequestRejectsDifferentTask(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	ctx, first := integrationTask(t, repo, domain.TaskStatusCreated)
	second, err := domain.NewTask("other", first.UserID, domain.TaskModeTrain, nil)
	require.NoError(t, err)
	second, err = repo.CreateTask(ctx, second)
	require.NoError(t, err)
	usecase := tasksRun.New(repo)
	require.NoError(t, usecase.RunTask(ctx, tasksRun.Input{TaskID: first.ID}))
	require.ErrorIs(t, usecase.RunTask(ctx, tasksRun.Input{TaskID: second.ID}), errs.ErrConflict)
	stored, err := repo.GetTaskByID(ctx, second.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusCreated, stored.Status)
	var events int
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&events))
	assert.Equal(t, 1, events)
}

func TestTaskIntegrationRunRequestScopesKeyToUser(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	firstCtx, first := integrationTask(t, repo, domain.TaskStatusCreated)
	secondCtx, second := integrationTask(t, repo, domain.TaskStatusCreated)
	key := core.MustIdempotencyKeyFromContext(firstCtx)
	secondCtx = key.WithContext(secondCtx)
	usecase := tasksRun.New(repo)
	require.NoError(t, usecase.RunTask(firstCtx, tasksRun.Input{TaskID: first.ID}))
	// The same text key cannot make one user's request look like another user's success.
	require.ErrorIs(t, usecase.RunTask(secondCtx, tasksRun.Input{TaskID: first.ID}), errs.ErrNotFound)
	require.NoError(t, usecase.RunTask(secondCtx, tasksRun.Input{TaskID: second.ID}))
	var events, requests int
	require.NoError(t, db.QueryRow(firstCtx, "SELECT COUNT(*) FROM events").Scan(&events))
	require.NoError(t, db.QueryRow(firstCtx, "SELECT COUNT(*) FROM task_run_requests").Scan(&requests))
	assert.Equal(t, 2, events)
	assert.Equal(t, 2, requests)
}

func TestTaskIntegrationConcurrentSameRunRequest(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	ctx, task := integrationTask(t, repo, domain.TaskStatusCreated)
	held := &heldTaskRepository{Repository: repo, locked: make(chan struct{}), release: make(chan struct{})}
	var once sync.Once
	release := func() {
		once.Do(func() {
			close(held.release)
		})
	}
	defer release()
	first := make(chan error, 1)
	second := make(chan error, 1)
	input := tasksRun.Input{TaskID: task.ID}
	go func() {
		first <- tasksRun.New(held).RunTask(ctx, input)
	}()
	awaitTaskLock(t, held)
	go func() {
		second <- tasksRun.New(repo).RunTask(ctx, input)
	}()
	waitTaskLock(t, db)
	release()
	require.NoError(t, <-first)
	require.NoError(t, <-second)
	var events, requests int
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&events))
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM task_run_requests").Scan(&requests))
	assert.Equal(t, 1, events)
	assert.Equal(t, 1, requests)
}

type failedEventRepository struct {
	*Repository
}

func (r failedEventRepository) PublishEvent(context.Context, domain.Event) (domain.Event, error) {
	return domain.Event{}, errors.New("outbox insert failed")
}

func TestTaskIntegrationRunRequestRollsBackWithEventFailure(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	ctx, task := integrationTask(t, repo, domain.TaskStatusCreated)
	input := tasksRun.Input{TaskID: task.ID}
	require.ErrorContains(t, tasksRun.New(failedEventRepository{repo}).RunTask(ctx, input), "outbox insert failed")
	stored, err := repo.GetTaskByID(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusCreated, stored.Status)
	_, err = repo.GetTaskRunRequest(ctx, task.UserID, core.MustIdempotencyKeyFromContext(ctx))
	require.ErrorIs(t, err, errs.ErrNotFound)
	require.NoError(t, tasksRun.New(repo).RunTask(ctx, input))
	var events int
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&events))
	assert.Equal(t, 1, events)
}

type ambiguousCommitRepository struct {
	*Repository
}

func (r ambiguousCommitRepository) InTransaction(ctx context.Context, fn func(context.Context) error) error {
	if err := r.Repository.InTransaction(ctx, fn); err != nil {
		return err
	}
	return errors.New("commit acknowledgement lost")
}

func TestTaskIntegrationRunRequestRecoversLostCommitAcknowledgement(t *testing.T) {
	repo, db := newTaskIntegrationRepository(t)
	ctx, task := integrationTask(t, repo, domain.TaskStatusCreated)
	input := tasksRun.Input{TaskID: task.ID}
	require.ErrorContains(t, tasksRun.New(ambiguousCommitRepository{repo}).RunTask(ctx, input), "acknowledgement lost")
	// A new usecase instance represents a retry after process restart; no Redis state is needed.
	require.NoError(t, tasksRun.New(repo).RunTask(ctx, input))
	var events, requests int
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM events").Scan(&events))
	require.NoError(t, db.QueryRow(ctx, "SELECT COUNT(*) FROM task_run_requests").Scan(&requests))
	assert.Equal(t, 1, events)
	assert.Equal(t, 1, requests)
}
