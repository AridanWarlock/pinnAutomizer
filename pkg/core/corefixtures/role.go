package corefixtures

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func NewRole(mods ...Mod[core.Role]) core.Role {
	us := core.Role{
		ID:    uuid.New(),
		Title: "ROLE_USER",
	}

	return Fixture(us, mods)
}
