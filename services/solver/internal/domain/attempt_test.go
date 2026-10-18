package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestAttemptRequiredFieldsAndMode(t *testing.T) {
	id, task := uuid.New(), uuid.New()
	now := time.Now()
	for _, in := range []struct {
		run, task uuid.UUID
		mode      string
		now       time.Time
	}{{uuid.Nil, task, "train", now}, {id, uuid.Nil, "train", now}, {id, task, "invalid", now}, {id, task, "train", time.Time{}}} {
		_, err := NewAttempt(in.run, in.task, in.mode, in.now)
		require.ErrorIs(t, err, errs.ErrInvalidArgument)
	}
	a, err := NewAttempt(id, task, "train", now)
	require.NoError(t, err)
	require.Equal(t, AttemptAccepted, a.State)
	require.Empty(t, a.ContainerID)
}

func TestEventRequiresOutgoingPayloadAndStableIdentity(t *testing.T) {
	id := uuid.New()
	message, err := core.NewProduceKafkaMessage("after-run", nil, []byte("result"), core.KafkaHeaders{core.KafkaIdempotencyHeader: id.String()})
	require.NoError(t, err)
	valid, err := NewEvent(id, uuid.New(), message)
	require.NoError(t, err)
	require.NoError(t, valid.Validate())
	for _, change := range []func(*Event){func(e *Event) {
		e.ID = uuid.Nil
	}, func(e *Event) {
		e.RunID = uuid.Nil
	}, func(e *Event) {
		e.Message.Topic = ""
	}, func(e *Event) {
		e.Message.Value = []byte{}
	}, func(e *Event) {
		e.Message.Headers = nil
	}, func(e *Event) {
		e.Message.Partition = 1
	}} {
		invalid := valid
		change(&invalid)
		require.ErrorIs(t, invalid.Validate(), errs.ErrInvalidArgument)
	}
}
