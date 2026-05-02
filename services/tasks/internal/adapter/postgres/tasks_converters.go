package postgres

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

type TaskRow struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`

	Mode domain.TaskMode `db:"mode"`

	Status domain.TaskStatus `db:"status"`
	Error  *string           `db:"error"`

	DataPath   string `db:"data_path"`
	OutputPath string `db:"output_path"`

	UserID    uuid.UUID `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
}

func (r TaskRow) Values() []any {
	return []any{
		r.ID,
		r.Name,
		r.Description,

		r.Mode,

		r.Status,
		r.Error,

		r.DataPath,
		r.OutputPath,

		r.UserID,
		r.CreatedAt,
	}
}

func ToTaskModel(r TaskRow) domain.Task {
	return domain.Task{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,

		Mode: r.Mode,

		Status: r.Status,
		Error:  r.Error,

		DataPath:   r.DataPath,
		OutputPath: r.OutputPath,

		UserID:    r.UserID,
		CreatedAt: r.CreatedAt,
	}
}

func FromTaskModel(t domain.Task) TaskRow {
	return TaskRow{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,

		Mode: t.Mode,

		Status: t.Status,
		Error:  t.Error,

		DataPath:   t.DataPath,
		OutputPath: t.OutputPath,

		UserID:    t.UserID,
		CreatedAt: t.CreatedAt,
	}
}
