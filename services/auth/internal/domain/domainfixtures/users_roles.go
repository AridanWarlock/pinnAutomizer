package domainfixtures

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
)

func NewUsersRoles(mods ...corefixtures.Mod[domain.UsersRoles]) domain.UsersRoles {
	ur := domain.UsersRoles{
		UserID: uuid.New(),
		RoleID: uuid.New(),
	}

	return corefixtures.Fixture(ur, mods)
}
