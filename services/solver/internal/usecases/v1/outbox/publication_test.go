package outbox

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func pendingEvent(t *testing.T) domain.Event {
	t.Helper()
	id := uuid.New()
	message, err := core.NewProduceKafkaMessage("tasks.after.run", []byte("task"), []byte(`{"result":"done"}`), nil)
	require.NoError(t, err)
	message, err = message.WithIdKey(core.IdempotencyKey(id.String()))
	require.NoError(t, err)
	event, err := domain.NewEvent(id, uuid.New(), message)
	require.NoError(t, err)
	return event
}

type partialFailure struct {
	error
	confirmed []int
}

func (e partialFailure) Confirmed() []int {
	return e.confirmed
}

func (e partialFailure) Unwrap() error {
	return e.error
}

func pendingSQLite(events ...domain.Event) *MockSQLite {
	return &MockSQLite{GetPendingEventsFunc: func(context.Context, limit.Limit) ([]domain.Event, error) {
		return events, nil
	}}
}

func TestPublishPendingReadFailureDoesNotSendOrDelete(t *testing.T) {
	t.Parallel()
	failure := errors.New("journal unavailable")
	sqlite := &MockSQLite{GetPendingEventsFunc: func(context.Context, limit.Limit) ([]domain.Event, error) {
		return nil, failure
	}}
	kafka := &MockKafka{}
	require.ErrorIs(t, New(sqlite, kafka).PublishPending(context.Background()), failure)
	require.Empty(t, kafka.WriteMessagesCalls())
	require.Empty(t, sqlite.DeleteEventsCalls())
}

func TestPublishPendingEmptyJournalDoesNotSendOrDelete(t *testing.T) {
	t.Parallel()
	sqlite := pendingSQLite()
	kafka := &MockKafka{}
	require.NoError(t, New(sqlite, kafka).PublishPending(context.Background()))
	require.Empty(t, kafka.WriteMessagesCalls())
	require.Empty(t, sqlite.DeleteEventsCalls())
}

func TestPublishPendingUnknownFailureKeepsEntireBatch(t *testing.T) {
	t.Parallel()
	first, second := pendingEvent(t), pendingEvent(t)
	failure := errors.New("confirmation lost")
	sqlite := pendingSQLite(first, second)
	kafka := &MockKafka{WriteMessagesFunc: func(context.Context, ...core.KafkaMessage) error {
		return failure
	}}
	require.ErrorIs(t, New(sqlite, kafka).PublishPending(context.Background()), failure)
	require.Len(t, kafka.WriteMessagesCalls(), 1)
	require.Equal(t, []core.KafkaMessage{first.Message, second.Message}, kafka.WriteMessagesCalls()[0].Messages)
	require.Empty(t, sqlite.DeleteEventsCalls())
}

func TestPublishPendingDeleteFailureRetriesIdenticalBatch(t *testing.T) {
	t.Parallel()
	first, second := pendingEvent(t), pendingEvent(t)
	failure := errors.New("acknowledgement could not be saved")
	sqlite := pendingSQLite(first, second)
	sqlite.DeleteEventsFunc = func(context.Context, []uuid.UUID) error {
		return failure
	}
	kafka := &MockKafka{WriteMessagesFunc: func(context.Context, ...core.KafkaMessage) error {
		return nil
	}}
	u := New(sqlite, kafka)
	require.ErrorIs(t, u.PublishPending(context.Background()), failure)
	require.Equal(t, []uuid.UUID{first.ID, second.ID}, sqlite.DeleteEventsCalls()[0].EventIDs)
	sqlite.DeleteEventsFunc = func(context.Context, []uuid.UUID) error {
		return nil
	}
	require.NoError(t, u.PublishPending(context.Background()))
	calls := kafka.WriteMessagesCalls()
	require.Len(t, calls, 2)
	require.Equal(t, calls[0].Messages, calls[1].Messages)
}

func TestPublishPendingAcknowledgesBatchAfterConfirmation(t *testing.T) {
	t.Parallel()
	first, second := pendingEvent(t), pendingEvent(t)
	var effects []string
	sqlite := pendingSQLite(first, second)
	sqlite.DeleteEventsFunc = func(_ context.Context, ids []uuid.UUID) error {
		require.Equal(t, []uuid.UUID{first.ID, second.ID}, ids)
		effects = append(effects, "delete")
		return nil
	}
	kafka := &MockKafka{WriteMessagesFunc: func(_ context.Context, messages ...core.KafkaMessage) error {
		require.Equal(t, []core.KafkaMessage{first.Message, second.Message}, messages)
		effects = append(effects, "publish")
		return nil
	}}
	require.NoError(t, New(sqlite, kafka).PublishPending(context.Background()))
	require.Equal(t, []string{"publish", "delete"}, effects)
}

func TestPublishPendingDeletesOnlyConfirmedPositions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		positions []int
		want      []int
	}{
		{"first confirmed", []int{0}, []int{0}},
		{"second confirmed", []int{1}, []int{1}},
		{"all failed", nil, nil},
		{"negative position", []int{-1}, nil},
		{"out of range", []int{0, 2}, nil},
		{"duplicate position", []int{0, 0}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			events := []domain.Event{pendingEvent(t), pendingEvent(t)}
			failure := errors.New("partial publication")
			sqlite := pendingSQLite(events...)
			sqlite.DeleteEventsFunc = func(_ context.Context, ids []uuid.UUID) error {
				var want []uuid.UUID
				for _, position := range tc.want {
					want = append(want, events[position].ID)
				}
				require.Equal(t, want, ids)
				return nil
			}
			kafka := &MockKafka{WriteMessagesFunc: func(context.Context, ...core.KafkaMessage) error {
				return fmt.Errorf("wrapped: %w", partialFailure{error: failure, confirmed: tc.positions})
			}}
			require.ErrorIs(t, New(sqlite, kafka).PublishPending(context.Background()), failure)
			if len(tc.want) == 0 {
				require.Empty(t, sqlite.DeleteEventsCalls())
			} else {
				require.Len(t, sqlite.DeleteEventsCalls(), 1)
			}
		})
	}
}

func TestPublishPendingPreservesPublishAndDeleteErrors(t *testing.T) {
	t.Parallel()
	publishErr, deleteErr := errors.New("partial publish"), errors.New("delete failed")
	sqlite := pendingSQLite(pendingEvent(t), pendingEvent(t))
	sqlite.DeleteEventsFunc = func(context.Context, []uuid.UUID) error {
		return deleteErr
	}
	kafka := &MockKafka{WriteMessagesFunc: func(context.Context, ...core.KafkaMessage) error {
		return partialFailure{error: publishErr, confirmed: []int{0}}
	}}
	err := New(sqlite, kafka).PublishPending(context.Background())
	require.ErrorIs(t, err, publishErr)
	require.ErrorIs(t, err, deleteErr)
}

func TestPublishPendingPassesCancellationToStorage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sqlite := &MockSQLite{GetPendingEventsFunc: func(ctx context.Context, _ limit.Limit) ([]domain.Event, error) {
		return nil, ctx.Err()
	}}
	kafka := &MockKafka{}
	require.ErrorIs(t, New(sqlite, kafka).PublishPending(ctx), context.Canceled)
	require.Empty(t, kafka.WriteMessagesCalls())
	require.Empty(t, sqlite.DeleteEventsCalls())
}

func TestPublishPendingCancellationDuringConfirmationKeepsBatch(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sqlite := pendingSQLite(pendingEvent(t))
	kafka := &MockKafka{WriteMessagesFunc: func(ctx context.Context, _ ...core.KafkaMessage) error {
		cancel()
		return ctx.Err()
	}}
	require.ErrorIs(t, New(sqlite, kafka).PublishPending(ctx), context.Canceled)
	require.Empty(t, sqlite.DeleteEventsCalls())
}
