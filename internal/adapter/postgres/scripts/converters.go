package scripts

import (
	"database/sql"
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
)

type ScriptRow struct {
	ID         uuid.UUID      `db:"id"`
	Filename   string         `db:"filename"`
	Path       string         `db:"path"`
	UploadTime sql.NullTime   `db:"upload_time"`
	Text       sql.NullString `db:"text"`
	UserID     uuid.UUID      `db:"user_id"`
}

func (r *ScriptRow) Values() []any {
	return []any{
		r.ID,
		r.Filename,
		r.Path,
		r.UploadTime,
		r.Text,
		r.UserID,
	}
}

func ToModel(r *ScriptRow) *domain.Script {
	if r == nil {
		return nil
	}

	return &domain.Script{
		ID:         r.ID,
		Filename:   r.Filename,
		Path:       r.Path,
		UploadTime: r.UploadTime.Time,
		Text:       r.Text.String,
		UserID:     r.UserID,
	}
}

func FromModel(s *domain.Script) ScriptRow {
	if s == nil {
		return ScriptRow{}
	}

	return ScriptRow{
		ID:         s.ID,
		Filename:   s.Filename,
		Path:       s.Path,
		UploadTime: sql.NullTime{Time: s.UploadTime, Valid: !s.UploadTime.IsZero()},
		Text:       sql.NullString{String: s.Text, Valid: s.Text != ""},
		UserID:     s.UserID,
	}
}
