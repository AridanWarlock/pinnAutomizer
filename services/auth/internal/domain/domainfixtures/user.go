package domainfixtures

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
)

func NewUser(mods ...corefixtures.Mod[domain.User]) domain.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)

	user := domain.User{
		ID:           uuid.New(),
		Login:        "Ivan Ivanov",
		PasswordHash: string(hash),
	}

	return corefixtures.Fixture(user, mods)
}
