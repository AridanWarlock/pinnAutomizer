package cleanupSessions

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

type Input struct {
	AfterJti *core.Jti
}

type Output struct {
	NextJti *core.Jti
}

type Usecase interface {
	Cleanup(ctx context.Context, in Input) (Output, error)
}
