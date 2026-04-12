package authMe

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

type Output struct {
	domain.User
}

type Usecase interface {
	Me(ctx context.Context) (Output, error)
}
