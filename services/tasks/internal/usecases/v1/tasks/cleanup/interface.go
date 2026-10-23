package tasksCleanup

import (
	"context"

	"github.com/google/uuid"
)

type Input struct {
	AfterTaskID uuid.UUID
}

type Output struct {
	// Nil marks the end of a sweep. Errors on individual jobs still advance the cursor.
	NextTaskID uuid.UUID
}

type Usecase interface {
	CleanupPending(ctx context.Context, in Input) (Output, error)
}
