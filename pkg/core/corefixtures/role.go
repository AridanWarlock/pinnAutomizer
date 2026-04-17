package corefixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewRole(mods ...Mod[core.Role]) core.Role {
	us := core.Role{
		ID:    uuid.New(),
		Title: "ROLE_USER",
	}

	return Fixture(us, mods)
}
