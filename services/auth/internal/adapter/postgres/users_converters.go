package postgres

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
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

func toUserModel(r UserRow) domain.User {
	return domain.User{
		ID:           r.ID,
		Login:        r.Login,
		PasswordHash: r.PasswordHash,
	}
}

func fromUserModel(u domain.User) UserRow {
	return UserRow{
		ID:           u.ID,
		Login:        u.Login,
		PasswordHash: u.PasswordHash,
	}
}
