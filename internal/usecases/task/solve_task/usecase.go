package solve_task

import (
	"context"
	"encoding/json"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/tx"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	GetTaskByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (domain.Task, error)
	PublishEvent(ctx context.Context, event domain.Event) error

	tx.Wrapper
}

type Usecase struct {
	postgres Postgres

	log zerolog.Logger
}

var usecase *Usecase

func New(postgres Postgres, log zerolog.Logger) *Usecase {
	usecase = &Usecase{
		postgres: postgres,

		log: log.With().Str("component", "usecase: task.SolveTask").Logger(),
	}

	return usecase
}

func (u *Usecase) SolveTask(ctx context.Context, in Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Error().Err(err).Msg("usecase: input.Validate")
		return err
	}

	task, err := u.postgres.GetTaskByIDAndUserID(ctx, in.TaskID, in.UserID)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.GetTaskByIDAndUserID")
		return err
	}

	event, err := u.createSolveTaskEvent(task)
	if err != nil {
		log.Error().Err(err).Msg("usecase: createSolveTaskEvent")
		return err
	}

	err = u.postgres.PublishEvent(ctx, event)
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.PublishEvent")
		return err
	}

	return nil
}

func (u *Usecase) createSolveTaskEvent(task domain.Task) (domain.Event, error) {
	msg := domain.SolveTaskMessage{
		TaskID:    task.ID,
		ModelPath: task.ResultsPath,
		Constants: task.Constants,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return domain.Event{}, err
	}

	return domain.NewEvent("to-solve", jsonMsg), nil
}
