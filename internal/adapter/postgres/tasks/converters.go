package tasks

import (
	"database/sql"
	"pinnAutomizer/internal/domain"
	"time"

	"github.com/google/uuid"
)

type TaskRow struct {
	ID               uuid.UUID      `db:"id"`
	Name             string         `db:"name"`
	Description      sql.NullString `db:"description"`
	Status           TaskStatus     `db:"status"`
	Constants        map[string]any `db:"constants"`
	TrainingDataPath sql.NullString `db:"training_data_path"`
	ResultsPath      sql.NullString `db:"results_path"`

	UserID     uuid.UUID `db:"user_id"`
	EquationID uuid.UUID `db:"equation_id"`

	CreatedAt time.Time `db:"created_at"`
}

func (r *TaskRow) Values() []any {
	return []any{
		r.ID,
		r.Name,
		r.Description,
		r.Status,
		r.Constants,
		r.TrainingDataPath,
		r.ResultsPath,

		r.UserID,
		r.EquationID,

		r.CreatedAt,
	}
}

func ToModel(r *TaskRow) domain.Task {
	if r == nil {
		return domain.Task{}
	}
	return domain.Task{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description.String,
		Status:      domain.TaskStatus(r.Status),
		Constants:   r.Constants,

		TrainingDataPath: r.TrainingDataPath.String,
		ResultsPath:      r.ResultsPath.String,

		UserID:     r.UserID,
		EquationID: r.EquationID,

		CreatedAt: r.CreatedAt,
	}
}

func FromModel(t domain.Task) TaskRow {
	return TaskRow{
		ID:          t.ID,
		Name:        t.Name,
		Description: sql.NullString{String: t.Description, Valid: t.Description != ""},
		Status:      TaskStatus(t.Status),
		Constants:   t.Constants,

		TrainingDataPath: sql.NullString{String: t.TrainingDataPath, Valid: t.TrainingDataPath != ""},
		ResultsPath:      sql.NullString{String: t.ResultsPath, Valid: t.ResultsPath != ""},
		UserID:           t.UserID,
		EquationID:       t.EquationID,

		CreatedAt: t.CreatedAt,
	}
}
