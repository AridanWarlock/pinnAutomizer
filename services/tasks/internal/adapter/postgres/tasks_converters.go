package postgres

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type TaskStatus string

func (s *TaskStatus) Scan(value any) error {
	if value == nil {
		return errors.New("scan nil value")
	}

	switch v := value.(type) {
	case []byte:
		*s = TaskStatus(v)
	case string:
		*s = TaskStatus(v)
	default:
		return fmt.Errorf("cannot scan %T into TaskStatus", value)
	}
	return nil
}

func (s TaskStatus) Value() (driver.Value, error) {
	return string(s), nil
}

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

func (r TaskRow) Values() []any {
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

func ToTaskModel(r TaskRow) domain.Task {
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

func FromTaskModel(t domain.Task) TaskRow {
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
