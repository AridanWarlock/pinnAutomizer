package fixtures

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type attemptInput struct {
	containerID    string
	startRequested bool
	now            time.Time
}

type AttemptOption func(*attemptInput)

func WithContainer(id string) AttemptOption {
	return func(in *attemptInput) {
		in.containerID = id
	}
}

func WithStartRequested() AttemptOption {
	return func(in *attemptInput) {
		in.startRequested = true
	}
}

// WithTime sets the fixture's creation and requested-start time.
func WithTime(now time.Time) AttemptOption {
	return func(in *attemptInput) { in.now = now }
}

func NewAttempt(t testing.TB, opts ...AttemptOption) domain.Attempt {
	t.Helper()
	in := attemptInput{now: time.Now()}
	for _, opt := range opts {
		opt(&in)
	}
	a, err := domain.NewAttempt(uuid.New(), uuid.New(), domain.MlTaskModePredict, in.now)
	if err != nil {
		t.Fatal(err)
	}
	if in.containerID != "" {
		if _, err := a.RecordContainer(in.containerID); err != nil {
			t.Fatal(err)
		}
	}
	if in.startRequested {
		if _, err := a.RequestStart(in.now, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	return a
}
