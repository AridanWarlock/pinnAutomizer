package update_script_after_translate

import (
	"context"
	"pinnAutomizer/internal/domain"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type Postgres interface {
	TranslateScript(ctx context.Context, in *domain.FromTranslate) error
}

var usecase *Usecase

type Usecase struct {
	postgres Postgres

	log      zerolog.Logger
	validate *validator.Validate
}

func New(
	postgres Postgres,
	log zerolog.Logger,
	validate *validator.Validate,
) *Usecase {
	uc := &Usecase{
		postgres: postgres,

		log: log.With().
			Str("component", "usecase: script.UpdateScriptAfterTranslate").
			Logger(),
		validate: validate,
	}

	usecase = uc

	return uc
}

func (u *Usecase) UpdateScriptAfterTranslate(ctx context.Context, input Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := input.Validate(u.validate); err != nil {
		log.Info().
			Err(err).
			Msg("input validation error")
		return err
	}

	err := u.postgres.TranslateScript(ctx, &domain.FromTranslate{
		ScriptID: input.ID,
		Text:     input.Text,
	})

	if err != nil {
		log.Error().
			Err(err).
			Msg("translate script postgres error")
		return err
	}

	return nil
}
