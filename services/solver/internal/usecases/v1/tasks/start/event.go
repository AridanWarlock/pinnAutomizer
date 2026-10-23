package tasksStart

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func newStartEvent(a domain.Attempt) (domain.Event, error) {
	payload, err := json.Marshal(struct {
		TaskID uuid.UUID `json:"task_id"`
	}{
		TaskID: a.TaskID,
	})
	if err != nil {
		return domain.Event{}, fmt.Errorf("marshal start event: %w", err)
	}
	id := uuid.NewSHA1(a.RunID, []byte("tasks.on.run"))
	message, err := core.NewProduceKafkaMessage(
		"tasks.on.run",
		[]byte(a.TaskID.String()),
		payload,
		core.KafkaHeaders{core.KafkaIdempotencyHeader: id.String()},
	)
	if err != nil {
		return domain.Event{}, err
	}
	return domain.NewEvent(id, a.RunID, message)
}
