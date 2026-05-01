package postgres

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

type RoleRaw struct {
	ID    uuid.UUID `db:"id"`
	Title string    `db:"title"`
}

func FromRoleModel(r core.Role) RoleRaw {
	return RoleRaw{
		ID:    r.ID,
		Title: r.Title,
	}
}

func ToRoleModel(r RoleRaw) core.Role {
	return core.Role{
		ID:    r.ID,
		Title: r.Title,
	}
}
