package sqlite

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestStartIntentAndEventRollbackThenConcurrentReplay(t *testing.T) {
	cfg := config(t)
	repo := open(t, cfg)
	a := fixtures.NewAttempt(t)
	_, _, err := repo.RegisterAttempt(t.Context(), a)
	require.NoError(t, err)
	_, err = repo.RecordContainer(t.Context(), a.RunID, "id")
	require.NoError(t, err)
	startEvent := fixtures.NewEvent(t, a)
	db := sqliteClient(repo).Database()
	_, err = db.Exec("CREATE TRIGGER reject_start BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT,'injected start event failure'); END;")
	require.NoError(t, err)
	changed, err := repo.RequestStart(t.Context(), a.RunID, time.Now(), time.Minute, startEvent, limit.MustNew(1))
	require.Error(t, err)
	require.Equal(t, core.Changed(false), changed)
	stored, err := repo.GetAttempt(t.Context(), a.RunID)
	require.NoError(t, err)
	require.Equal(t, domain.AttemptContainerCreated, stored.State)
	require.Nil(t, stored.Deadline, "failed event insertion must not consume the one permitted Start")
	events, err := repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, events)
	_, err = db.Exec("DROP TRIGGER reject_start")
	require.NoError(t, err)

	foreign := fixtures.NewEvent(t, fixtures.NewAttempt(t))
	_, err = repo.RequestStart(t.Context(), a.RunID, time.Now(), time.Minute, foreign, limit.MustNew(1))
	require.ErrorIs(t, err, errs.ErrInvalidArgument)

	type result struct {
		changed core.Changed
		err     error
	}
	results := make(chan result, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() {
			changed, err := repo.RequestStart(t.Context(), a.RunID, time.Now(), time.Minute, startEvent, limit.MustNew(1))
			results <- result{changed: changed, err: err}
		})
	}
	workers.Wait()
	close(results)
	changes := 0
	for result := range results {
		require.NoError(t, result.err)
		if result.changed {
			changes++
		}
	}
	require.Equal(t, 1, changes, "one caller receives permission to send Docker Start")
	events, err = repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Equal(t, []domain.Event{startEvent}, events)
	first, err := repo.GetAttempt(t.Context(), a.RunID)
	require.NoError(t, err)
	require.NoError(t, repo.DeleteEvents(t.Context(), []uuid.UUID{startEvent.ID}))
	require.NoError(t, repo.Close())
	repo = open(t, cfg)
	changed, err = repo.RequestStart(t.Context(), a.RunID, time.Now(), time.Hour, fixtures.NewEvent(t, a), limit.MustNew(1))
	require.NoError(t, err)
	require.Equal(t, core.Changed(false), changed)
	stored, err = repo.GetAttempt(t.Context(), a.RunID)
	require.NoError(t, err)
	require.Equal(t, first, stored, "replay must preserve the original deadline")
	events, err = repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, events, "replay after delivery must not recreate the start event")
}
