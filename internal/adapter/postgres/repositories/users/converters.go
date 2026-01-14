package users

import (
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
)

type UserRow struct {
	ID           uuid.UUID `db:"id"`
	Login        string    `db:"login"`
	PasswordHash string    `db:"password_hash"`
}

func (r UserRow) Values() []any {
	return []any{
		r.ID,
		r.Login,
		r.PasswordHash,
	}
}

func ToModel(r UserRow) domain.User {
	return domain.User{
		ID:           r.ID,
		Login:        r.Login,
		PasswordHash: r.PasswordHash,
	}
}

func FromModel(u domain.User) UserRow {
	return UserRow{
		ID:           u.ID,
		Login:        u.Login,
		PasswordHash: u.PasswordHash,
	}
}
