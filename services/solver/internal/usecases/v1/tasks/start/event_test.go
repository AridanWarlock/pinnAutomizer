package tasksStart

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestStartEventPreservesWireContractAndStableIdentity(t *testing.T) {
	t.Parallel()
	a := fixtures.NewAttempt(t)
	event, err := newStartEvent(a)
	require.NoError(t, err)
	replay, err := newStartEvent(a)
	require.NoError(t, err)
	require.Equal(t, event, replay)
	require.Equal(t, a.RunID, event.RunID)
	require.Equal(t, "tasks.on.run", event.Message.Topic)
	require.Equal(t, []byte(a.TaskID.String()), event.Message.Key)
	require.JSONEq(t, fmt.Sprintf(`{"task_id":%q}`, a.TaskID.String()), string(event.Message.Value))
	require.Equal(t, event.ID.String(), event.Message.Headers[core.KafkaIdempotencyHeader])
	require.NotEqual(t, uuid.NewSHA1(a.RunID, []byte("tasks.after.run")), event.ID, "start and completion need distinct delivery identities")
	other, err := newStartEvent(fixtures.NewAttempt(t))
	require.NoError(t, err)
	require.NotEqual(t, event.ID, other.ID)
}
