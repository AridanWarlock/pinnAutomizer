package tasksAccept

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func TestAcceptTaskDurableReplayAndValidation(t *testing.T) {
	ctx := context.Background()
	cfg := sqlite.Config{Path: filepath.Join(t.TempDir(), "journal.db"), BusyTimeout: time.Second}
	dir, err := filepath.Abs("../../../../../migrations")
	require.NoError(t, err)
	cfg.MigrationsDir = dir
	db, err := sql.Open("sqlite", cfg.Path)
	require.NoError(t, err)
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(dir))
	require.NoError(t, err)
	_, err = provider.Up(context.Background())
	require.NoError(t, err)
	require.NoError(t, db.Close())
	repo, err := sqlite.New(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.Close()
	})
	u := New(repo)
	in := Input{RunID: uuid.New(), TaskID: uuid.New(), Mode: "train"}
	first, err := u.AcceptTask(ctx, in)
	require.NoError(t, err)
	require.True(t, first.Created)
	require.NoError(t, repo.Close())
	repo, err = sqlite.New(cfg)
	require.NoError(t, err)
	u = New(repo)
	replay, err := u.AcceptTask(ctx, in)
	require.NoError(t, err)
	require.False(t, replay.Created)
	require.Equal(t, first.Attempt, replay.Attempt)
	changed := in
	changed.Mode = "predict"
	_, err = u.AcceptTask(ctx, changed)
	require.ErrorIs(t, err, errs.ErrConflict)
	for _, invalid := range []Input{{TaskID: in.TaskID, Mode: "train"}, {RunID: in.RunID, Mode: "train"}, {RunID: in.RunID, TaskID: in.TaskID, Mode: "invalid"}} {
		_, err = u.AcceptTask(ctx, invalid)
		require.ErrorIs(t, err, errs.ErrInvalidArgument)
	}
	require.NoError(t, repo.Close())
	_, err = u.AcceptTask(ctx, in)
	require.Error(t, err)
}

func TestConsumerPropagatesStorageFailure(t *testing.T) {
	failure := errors.New("disk full")
	repo := &MockSQLite{RegisterAttemptFunc: func(context.Context, domain.Attempt) (domain.Attempt, bool, error) {
		return domain.Attempt{}, false, failure
	}}
	c := NewConsumer(New(repo))
	msg := core.KafkaMessage{Headers: core.KafkaHeaders{"X-Idempotency-Key": uuid.NewString()}, Value: []byte(fmt.Sprintf(`{"task_id":%q,"mode":"train"}`, uuid.NewString()))}
	require.ErrorIs(t, c.HandleMessage(context.Background(), msg), failure)
	require.Len(t, repo.RegisterAttemptCalls(), 1)
	msg.Headers = nil
	require.ErrorIs(t, c.HandleMessage(context.Background(), msg), errs.ErrInvalidArgument)
	require.Len(t, repo.RegisterAttemptCalls(), 1)
	msg.Headers = core.KafkaHeaders{"X-Idempotency-Key": uuid.NewString()}
	msg.Value = []byte(`{"mode":"invalid"}`)
	require.ErrorIs(t, c.HandleMessage(context.Background(), msg), errs.ErrInvalidArgument)
	require.Len(t, repo.RegisterAttemptCalls(), 1)
}
