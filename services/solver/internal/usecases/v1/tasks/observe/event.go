package tasksObserve

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func newCompletionEvent(a domain.Attempt) (domain.Event, error) {
	payload, err := json.Marshal(struct {
		TaskID uuid.UUID `json:"task_id"`
		Error  *string   `json:"error,omitempty"`
	}{
		TaskID: a.TaskID,
		Error:  a.Error,
	})
	if err != nil {
		return domain.Event{}, fmt.Errorf("marshal completion: %w", err)
	}
	// One logical completion per attempt, independent of observation/retry time.
	id := uuid.NewSHA1(a.RunID, []byte("tasks.after.run"))
	message, err := core.NewProduceKafkaMessage(
		"tasks.after.run",
		[]byte(a.TaskID.String()),
		payload,
		core.KafkaHeaders{core.KafkaIdempotencyHeader: id.String()},
	)
	if err != nil {
		return domain.Event{}, err
	}
	return domain.NewEvent(id, a.RunID, message)
}
