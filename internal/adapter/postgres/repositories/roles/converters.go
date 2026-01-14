package roles

import (
	"pinnAutomizer/internal/domain"

	"github.com/google/uuid"
)

type RoleRaw struct {
	ID    uuid.UUID `db:"id"`
	Title string    `db:"title"`
}

func FromModel(r domain.Role) RoleRaw {
	return RoleRaw{
		ID:    r.ID,
		Title: r.Title,
	}
}

func ToModel(r RoleRaw) domain.Role {
	return domain.Role{
		ID:    r.ID,
		Title: r.Title,
	}
}
