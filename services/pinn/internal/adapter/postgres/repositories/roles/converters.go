package roles

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"

	"github.com/google/uuid"
)

type RoleRaw struct {
	ID    uuid.UUID `db:"id"`
	Title string    `db:"title"`
}

func FromModel(r core.Role) RoleRaw {
	return RoleRaw{
		ID:    r.ID,
		Title: r.Title,
	}
}

func ToModel(r RoleRaw) core.Role {
	return core.Role{
		ID:    r.ID,
		Title: r.Title,
	}
}
