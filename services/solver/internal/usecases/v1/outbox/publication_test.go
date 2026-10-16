package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func pendingEvent() domain.Event {
	id := uuid.New()
	return domain.Event{
		ID:    id,
		RunID: uuid.New(),
		Message: core.KafkaMessage{
			Topic:   "tasks.after.run",
			Key:     []byte("task"),
			Value:   []byte(`{"result":"done"}`),
			Headers: core.KafkaHeaders{"X-Idempotency-Key": id.String()},
		},
	}
}

func TestPublishPendingReadFailureDoesNotSendOrDelete(t *testing.T) {
	t.Parallel()
	failure := errors.New("journal unavailable")
	sqlite := &MockSQLite{GetPendingEventsFunc: func(context.Context, int) ([]domain.Event, error) {
		return nil, failure
	}}
	kafka := &MockKafka{}
	err := New(sqlite, kafka).PublishPending(context.Background())
	require.ErrorIs(t, err, failure)
	require.Empty(t, kafka.WriteMessagesCalls())
	require.Empty(t, sqlite.DeleteEventCalls())
}

func TestPublishPendingEmptyJournalDoesNotSendOrDelete(t *testing.T) {
	t.Parallel()
	sqlite := &MockSQLite{GetPendingEventsFunc: func(context.Context, int) ([]domain.Event, error) {
		return nil, nil
	}}
	kafka := &MockKafka{}
	require.NoError(t, New(sqlite, kafka).PublishPending(context.Background()))
	require.Empty(t, kafka.WriteMessagesCalls())
	require.Empty(t, sqlite.DeleteEventCalls())
}

func TestPublishPendingStopsAtFirstUnconfirmedMessage(t *testing.T) {
	t.Parallel()
	first, second := pendingEvent(), pendingEvent()
	failure := errors.New("broker did not confirm")
	sqlite := &MockSQLite{GetPendingEventsFunc: func(context.Context, int) ([]domain.Event, error) {
		return []domain.Event{first, second}, nil
	}}
	kafka := &MockKafka{WriteMessagesFunc: func(context.Context, ...core.KafkaMessage) error {
		return failure
	}}
	err := New(sqlite, kafka).PublishPending(context.Background())
	require.ErrorIs(t, err, failure)
	require.Len(t, kafka.WriteMessagesCalls(), 1)
	require.Equal(t, []core.KafkaMessage{first.Message}, kafka.WriteMessagesCalls()[0].Messages)
	require.Empty(t, sqlite.DeleteEventCalls())
}

func TestPublishPendingDeleteFailureRetriesIdenticalMessage(t *testing.T) {
	t.Parallel()
	first, second := pendingEvent(), pendingEvent()
	failure := errors.New("acknowledgement could not be saved")
	deleteFails := true
	sqlite := &MockSQLite{
		GetPendingEventsFunc: func(context.Context, int) ([]domain.Event, error) {
			return []domain.Event{first, second}, nil
		},
		DeleteEventFunc: func(_ context.Context, id uuid.UUID) error {
			if deleteFails {
				return failure
			}
			return nil
		},
	}
	kafka := &MockKafka{WriteMessagesFunc: func(context.Context, ...core.KafkaMessage) error {
		return nil
	}}
	u := New(sqlite, kafka)
	require.ErrorIs(t, u.PublishPending(context.Background()), failure)
	require.Len(t, kafka.WriteMessagesCalls(), 1)
	require.Len(t, sqlite.DeleteEventCalls(), 1)
	require.Equal(t, first.ID, sqlite.DeleteEventCalls()[0].EventID)

	deleteFails = false
	require.NoError(t, u.PublishPending(context.Background()))
	calls := kafka.WriteMessagesCalls()
	require.Len(t, calls, 3)
	require.Equal(t, []core.KafkaMessage{first.Message}, calls[0].Messages)
	require.Equal(t, calls[0].Messages, calls[1].Messages)
	require.Equal(t, []core.KafkaMessage{second.Message}, calls[2].Messages)
}

func TestPublishPendingAcknowledgesEachMessageAfterConfirmation(t *testing.T) {
	t.Parallel()
	first, second := pendingEvent(), pendingEvent()
	var effects []string
	sqlite := &MockSQLite{
		GetPendingEventsFunc: func(context.Context, int) ([]domain.Event, error) {
			return []domain.Event{first, second}, nil
		},
		DeleteEventFunc: func(_ context.Context, id uuid.UUID) error {
			effects = append(effects, "delete "+id.String())
			return nil
		},
	}
	kafka := &MockKafka{WriteMessagesFunc: func(_ context.Context, messages ...core.KafkaMessage) error {
		require.Len(t, messages, 1)
		effects = append(effects, "publish "+messages[0].Headers["X-Idempotency-Key"])
		return nil
	}}
	require.NoError(t, New(sqlite, kafka).PublishPending(context.Background()))
	require.Equal(t, []string{
		"publish " + first.ID.String(), "delete " + first.ID.String(),
		"publish " + second.ID.String(), "delete " + second.ID.String(),
	}, effects)
}

func TestPublishPendingPassesCancellationToStorage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sqlite := &MockSQLite{GetPendingEventsFunc: func(ctx context.Context, _ int) ([]domain.Event, error) {
		return nil, ctx.Err()
	}}
	kafka := &MockKafka{}
	err := New(sqlite, kafka).PublishPending(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, kafka.WriteMessagesCalls())
	require.Empty(t, sqlite.DeleteEventCalls())
}

func TestPublishPendingCancellationDuringConfirmationKeepsMessage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sqlite := &MockSQLite{GetPendingEventsFunc: func(context.Context, int) ([]domain.Event, error) {
		return []domain.Event{pendingEvent()}, nil
	}}
	kafka := &MockKafka{WriteMessagesFunc: func(ctx context.Context, _ ...core.KafkaMessage) error {
		cancel()
		return ctx.Err()
	}}
	err := New(sqlite, kafka).PublishPending(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, kafka.WriteMessagesCalls(), 1)
	require.Empty(t, sqlite.DeleteEventCalls())
}
