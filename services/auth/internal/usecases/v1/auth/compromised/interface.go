package authCompromised

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

type Input struct {
	Jti core.Jti
}

type Usecase interface {
	HandleCompromisedSession(ctx context.Context, in Input) error
}
