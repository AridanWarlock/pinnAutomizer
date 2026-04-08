package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/google/uuid"
)

func NewRole(mods ...mod[domain.Role]) domain.Role {
	us := domain.Role{
		ID:    uuid.New(),
		Title: "ROLE_USER",
	}

	return fixture(us, mods)
}
