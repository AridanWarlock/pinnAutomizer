package fixtures

import (
	"testing"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

// NewEvent supplies an outgoing fixture, independent of business wire contracts.
func NewEvent(t testing.TB, a domain.Attempt) domain.Event {
	t.Helper()
	id := uuid.New()
	msg, err := core.NewProduceKafkaMessage("fixture.start", []byte(a.TaskID.String()), []byte("{}"),
		core.KafkaHeaders{core.KafkaIdempotencyHeader: id.String()})
	if err != nil {
		t.Fatal(err)
	}
	event, err := domain.NewEvent(id, a.RunID, msg)
	if err != nil {
		t.Fatal(err)
	}
	return event
}
