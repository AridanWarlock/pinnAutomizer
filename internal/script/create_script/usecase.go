package create_script

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/domain"

	"github.com/rs/zerolog"
)

type Postgres interface {
	CreateScript(ctx context.Context, in *domain.Script) (*domain.Script, error)
}

type Translator interface {
	Translate(ctx context.Context, translate *domain.ToTranslate) error
}

type Usecase struct {
	postgres   Postgres
	translator Translator

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	translator Translator,
	log zerolog.Logger,
) *Usecase {
	uc := &Usecase{
		postgres:   postgres,
		translator: translator,

		log: log.With().Str("component", "usecase: script.CreateScript").Logger(),
	}

	usecase = uc //global for handlers

	return uc
}

func (u *Usecase) CreateScript(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().
			Err(err).
			Msg("input validation error")
		return Output{}, err
	}

	script, err := domain.NewScript(in.Filename, in.Path, in.UserID)

	script, err = u.postgres.CreateScript(ctx, script)
	if err != nil {
		log.Error().
			Err(err).
			Msg("insert script postgres error")
		return Output{}, fmt.Errorf("create script in postgres: %w", err)
	}

	go func() {
		ctx := context.TODO()
		log := log.With().Ctx(ctx).Logger()

		err := u.translator.Translate(ctx, &domain.ToTranslate{
			ID:   script.ID,
			Path: script.Path,
		})

		if err != nil {
			log.Error().
				Err(err).
				Msg("translate script error")
		}
	}()

	return Output{
		ID:       script.ID,
		Filename: script.Filename,
		Text:     script.Text,
	}, nil
}
