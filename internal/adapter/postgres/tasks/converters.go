package tasks

import (
	"database/sql"

	"github.com/google/uuid"
)

type TaskRow struct {
	ID          uuid.UUID      `db:"id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	Status      TaskStatus     `db:"status"`
	Constants   string         `db:"constants"`
}
