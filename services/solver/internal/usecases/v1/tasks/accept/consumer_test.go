package tasksAccept

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func TestConsumerMapsCommandAndRunIdentity(t *testing.T) {
	t.Parallel()
	runID, taskID := uuid.New(), uuid.New()
	body, err := json.Marshal(Message{TaskID: taskID, Mode: "predict"})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	usecase := &MockUsecase{AcceptTaskFunc: func(received context.Context, in Input) (Output, error) {
		require.Equal(t, ctx, received)
		require.Equal(t, Input{RunID: runID, TaskID: taskID, Mode: "predict"}, in)
		return Output{}, nil
	}}
	msg := core.KafkaMessage{
		Headers: core.KafkaHeaders{"X-Idempotency-Key": runID.String()},
		Value:   body,
	}
	require.NoError(t, NewConsumer(usecase).HandleMessage(ctx, msg))
	require.Len(t, usecase.AcceptTaskCalls(), 1)
}

func TestConsumerRejectsMalformedEnvelopeBeforeUsecase(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers core.KafkaHeaders
		body    string
	}{
		{name: "missing run ID", body: `{}`},
		{name: "malformed run ID", headers: core.KafkaHeaders{"X-Idempotency-Key": "not-a-uuid"}, body: `{}`},
		{name: "invalid JSON", headers: core.KafkaHeaders{"X-Idempotency-Key": uuid.NewString()}, body: `{"task_id":`},
		{name: "invalid task UUID", headers: core.KafkaHeaders{"X-Idempotency-Key": uuid.NewString()}, body: `{"task_id":"not-a-uuid"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			usecase := &MockUsecase{}
			msg := core.KafkaMessage{Headers: tc.headers, Value: []byte(tc.body)}
			err := NewConsumer(usecase).HandleMessage(context.Background(), msg)
			require.ErrorIs(t, err, errs.ErrInvalidArgument)
			require.Empty(t, usecase.AcceptTaskCalls())
		})
	}
}

func TestConsumerReturnsAcceptanceFailure(t *testing.T) {
	t.Parallel()
	failure := errors.New("command was not persisted")
	usecase := &MockUsecase{AcceptTaskFunc: func(context.Context, Input) (Output, error) {
		return Output{}, failure
	}}
	msg := core.KafkaMessage{
		Headers: core.KafkaHeaders{"X-Idempotency-Key": uuid.NewString()},
		Value:   []byte(`{"task_id":"` + uuid.NewString() + `","mode":"train"}`),
	}
	require.ErrorIs(t, NewConsumer(usecase).HandleMessage(context.Background(), msg), failure)
}

func TestAcceptTaskReturnsPersistedAttemptOnReplay(t *testing.T) {
	t.Parallel()
	stored, err := domain.NewAttempt(uuid.New(), uuid.New(), "train", time.Unix(100, 0))
	require.NoError(t, err)
	_, err = stored.RecordContainer("existing-container")
	require.NoError(t, err)
	sqlite := &MockSQLite{RegisterAttemptFunc: func(_ context.Context, candidate domain.Attempt) (domain.Attempt, core.Created, error) {
		require.True(t, stored.Matches(candidate))
		return stored, false, nil
	}}
	out, err := New(sqlite).AcceptTask(context.Background(), Input{RunID: stored.RunID, TaskID: stored.TaskID, Mode: stored.Mode})
	require.NoError(t, err)
	require.False(t, out.Created)
	require.Equal(t, stored, out.Attempt)
}

func TestAcceptTaskInvalidInputDoesNotAccessJournal(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input Input
	}{
		{name: "missing run ID", input: Input{TaskID: uuid.New(), Mode: "train"}},
		{name: "missing task ID", input: Input{RunID: uuid.New(), Mode: "train"}},
		{name: "unknown mode", input: Input{RunID: uuid.New(), TaskID: uuid.New(), Mode: "unknown"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sqlite := &MockSQLite{}
			_, err := New(sqlite).AcceptTask(context.Background(), tc.input)
			require.ErrorIs(t, err, errs.ErrInvalidArgument)
			require.Empty(t, sqlite.RegisterAttemptCalls())
		})
	}
}

func TestAcceptTaskReturnsJournalFailuresWithoutOutput(t *testing.T) {
	for _, failure := range []error{errs.ErrConflict, context.Canceled} {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()
			sqlite := &MockSQLite{RegisterAttemptFunc: func(context.Context, domain.Attempt) (domain.Attempt, core.Created, error) {
				return domain.Attempt{}, false, failure
			}}
			out, err := New(sqlite).AcceptTask(context.Background(), Input{RunID: uuid.New(), TaskID: uuid.New(), Mode: "train"})
			require.ErrorIs(t, err, failure)
			require.Equal(t, Output{}, out)
		})
	}
}
