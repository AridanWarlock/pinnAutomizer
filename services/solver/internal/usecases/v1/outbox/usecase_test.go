package outbox

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	worker "github.com/AridanWarlock/pinnAutomizer/solver/internal/outbox"
)

type recordingWriter struct {
	messages []core.KafkaMessage
	err      error
	entered  chan struct {
	}
	failAfter int
}

func (w *recordingWriter) WriteMessages(ctx context.Context, messages ...core.KafkaMessage) error {
	if w.entered != nil {
		close(w.entered)
		<-ctx.Done()
		return ctx.Err()
	}
	w.messages = append(w.messages, messages...)
	if w.err != nil && w.failAfter > 0 {
		return partialFailure{error: w.err, confirmed: []int{0}}
	}
	return w.err
}

func setup(t *testing.T) (*sqlite.Repository, sqlite.Config, domain.Event) {
	t.Helper()
	cfg := sqlite.Config{Path: filepath.Join(t.TempDir(), "journal.db"), BusyTimeout: time.Second}
	dir, err := filepath.Abs("../../../../migrations")
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
	a, err := domain.NewAttempt(uuid.New(), uuid.New(), "train", time.Now())
	require.NoError(t, err)
	_, _, err = repo.RegisterAttempt(context.Background(), a)
	require.NoError(t, err)
	id := uuid.New()
	message, err := core.NewProduceKafkaMessage("after-run", []byte(a.TaskID.String()), []byte("done"), core.KafkaHeaders{core.KafkaIdempotencyHeader: id.String()})
	require.NoError(t, err)
	e, err := domain.NewEvent(id, a.RunID, message)
	require.NoError(t, err)
	code := 0
	_, err = repo.FinishAttempt(context.Background(), a.RunID, &code, nil, time.Now(), e)
	require.NoError(t, err)
	return repo, cfg, e
}

func TestAmbiguousPublishIsRetriedAfterRestart(t *testing.T) {
	ctx := context.Background()
	repo, cfg, e := setup(t)
	defer func() {
		_ = repo.Close()
	}()
	writer := &recordingWriter{err: errors.New("confirmation lost")}
	u := New(repo, writer)
	require.Error(t, u.PublishPending(ctx))
	events, err := repo.GetPendingEvents(ctx, limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.NoError(t, repo.Close())
	repo, err = sqlite.New(cfg)
	require.NoError(t, err)
	writer.err = nil
	u = New(repo, writer)
	require.NoError(t, u.PublishPending(ctx))
	require.Equal(t, []core.KafkaMessage{e.Message, e.Message}, writer.messages)
	events, err = repo.GetPendingEvents(ctx, limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestShutdownWaitsForPublishAndKeepsEvent(t *testing.T) {
	repo, _, _ := setup(t)
	defer repo.Close()
	writer := &recordingWriter{entered: make(chan struct {
	})}
	u := New(repo, writer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	publisher, err := worker.NewWorker(u, time.Hour, zerolog.Nop())
	require.NoError(t, err)
	go func() {
		done <- publisher.Run(ctx)
	}()
	select {
	case <-writer.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("publisher did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("publisher did not stop")
	}
	events, err := repo.GetPendingEvents(context.Background(), limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, events, 1)
}

func TestPartialPublicationKeepsRemainingEvent(t *testing.T) {
	repo, _, first := setup(t)
	defer repo.Close()
	ctx := context.Background()
	a, err := domain.NewAttempt(uuid.New(), uuid.New(), "train", time.Now())
	require.NoError(t, err)
	_, _, err = repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	id := uuid.New()
	message, err := first.Message.WithIdKey(core.IdempotencyKey(id.String()))
	require.NoError(t, err)
	second, err := domain.NewEvent(id, a.RunID, message)
	require.NoError(t, err)
	code := 0
	_, err = repo.FinishAttempt(ctx, a.RunID, &code, nil, time.Now(), second)
	require.NoError(t, err)
	writer := &recordingWriter{err: errors.New("broker unavailable"), failAfter: 2}
	u := New(repo, writer)
	require.Error(t, u.PublishPending(ctx))
	remaining, err := repo.GetPendingEvents(ctx, limit.MustNew(10))
	require.NoError(t, err)
	require.Equal(t, []domain.Event{second}, remaining)
	writer.err = nil
	require.NoError(t, u.PublishPending(ctx))
	require.Equal(t, []core.KafkaMessage{first.Message, second.Message, second.Message}, writer.messages)
}
