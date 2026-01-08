package search_scripts

import (
	"context"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/internal/domain/pagination"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	SearchScripts(
		ctx context.Context,
		userID uuid.UUID,
		f *domain.ScriptFilter,
		opts pagination.Options,
	) ([]*domain.Script, error)
}

type Usecase struct {
	postgres Postgres

	log      zerolog.Logger
	validate *validator.Validate
}

var usecase *Usecase

func New(
	postgres Postgres,
	log zerolog.Logger,
	validate *validator.Validate,
) *Usecase {
	uc := &Usecase{
		postgres: postgres,

		log:      log.With().Str("component", "usecase: script.SearchScripts").Logger(),
		validate: validate,
	}

	usecase = uc //global for handlers

	return uc
}

func (u *Usecase) SearchScripts(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(u.validate); err != nil {
		log.Info().
			Err(err).
			Msg("input validation error")
		return Output{}, err
	}

	scripts, err := u.postgres.SearchScripts(ctx, in.userID, in.f, in.p)
	if err != nil {
		log.Error().
			Err(err).
			Msg("search scripts in postgres error")

		return Output{}, err
	}

	return Output{
		Scripts: scripts,
		Count:   len(scripts),
	}, err
}
