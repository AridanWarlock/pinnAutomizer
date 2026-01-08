package create_user

import (
	"database/sql"
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
)

type CreateUserRow struct {
	ID           uuid.UUID `db:"id"`
	Login        string    `db:"login"`
	PasswordHash string    `db:"password_hash"`
}

func (r *CreateUserRow) Values() []any {
	return []any{
		r.ID,
		r.Login,
		r.PasswordHash,
	}
}

func (r *CreateUserRow) ToModel() *domain.User {
	if r == nil {
		return nil
	}

	return &domain.User{
		ID:           r.ID,
		Login:        r.Login,
		PasswordHash: r.PasswordHash,
	}
}

func FromModel(u *domain.User) *CreateUserRow {
	if u == nil {
		return nil
	}

	return &CreateUserRow{
		ID:           u.ID,
		Login:        u.Login,
		PasswordHash: u.PasswordHash,
	}
}

type AuthTokenRow struct {
	UserID       uuid.UUID      `db:"user_id"`
	AccessToken  sql.NullString `db:"access_token"`
	RefreshToken sql.NullString `db:"refresh_token"`
}

func (r *AuthTokenRow) Values() []any {
	return []any{
		r.UserID,
		r.AccessToken,
		r.RefreshToken,
	}
}
