package postgres

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
)

type UsersRolesRow struct {
	UserID uuid.UUID `db:"user_id"`
	RoleID uuid.UUID `db:"role_id"`
}

func (r UsersRolesRow) Values() []any {
	return []any{
		r.UserID,
		r.RoleID,
	}
}

func fromUsersRolesModel(ur domain.UsersRoles) UsersRolesRow {
	return UsersRolesRow{
		UserID: ur.UserID,
		RoleID: ur.RoleID,
	}
}

func toUsersRolesModel(r UsersRolesRow) domain.UsersRoles {
	return domain.UsersRoles{
		UserID: r.UserID,
		RoleID: r.RoleID,
	}
}
