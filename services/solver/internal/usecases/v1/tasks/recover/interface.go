package tasksRecover

import (
	"context"

	"github.com/google/uuid"
)

type Input struct {
	AfterRunID uuid.UUID
}

type Output struct {
	// NextRunID advances even past failed attempts. Nil marks the end of a sweep.
	// On a listing failure it retains the input cursor for retry.
	NextRunID uuid.UUID
}

type Usecase interface {
	RecoverPending(ctx context.Context, in Input) (Output, error)
}
