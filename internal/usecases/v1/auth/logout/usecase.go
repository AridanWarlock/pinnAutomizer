package authLogout

import (
	"context"
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Postgres interface {
	Logout(ctx context.Context, userID uuid.UUID, fingerprint domain.Fingerprint) error
}

type usecase struct {
	postgres Postgres
}

func New(
	postgres Postgres,
) Usecase {
	return &usecase{
		postgres: postgres,
	}
}

func (u *usecase) Logout(ctx context.Context, in Input) error {
	log := logger.FromContext(ctx)
	if err := in.Validate(); err != nil {
		return fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	if err := u.postgres.Logout(ctx, in.UserID, in.Fingerprint); err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			log.Info().Err(err).Msg("session already deleted")
			return nil
		}
		return fmt.Errorf("delete session from postgres: %w", err)
	}
	return nil
}
