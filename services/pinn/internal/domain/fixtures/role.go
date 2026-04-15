package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewRole(mods ...mod[core.Role]) core.Role {
	us := core.Role{
		ID:    uuid.New(),
		Title: "ROLE_USER",
	}

	return fixture(us, mods)
}
