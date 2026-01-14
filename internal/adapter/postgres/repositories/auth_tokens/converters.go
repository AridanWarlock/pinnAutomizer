package auth_tokens

import (
	"database/sql"
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
)

type AuthTokenRow struct {
	UserID       uuid.UUID      `db:"user_id"`
	AccessToken  sql.NullString `db:"access_token"`
	RefreshToken sql.NullString `db:"refresh_token"`
}

func (r AuthTokenRow) Values() []any {
	return []any{
		r.UserID,
		r.AccessToken,
		r.RefreshToken,
	}
}

func ToModel(r AuthTokenRow) domain.AuthToken {
	return domain.AuthToken{
		UserID:       r.UserID,
		AccessToken:  r.AccessToken.String,
		RefreshToken: r.RefreshToken.String,
	}
}
