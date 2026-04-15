package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
	"github.com/google/uuid"
)

func NewUser(mods ...mod[domain.User]) domain.User {
	us := domain.User{
		ID:           uuid.New(),
		Login:        "Ivan Ivanov",
		PasswordHash: "password_hash",
	}

	return fixture(us, mods)
}
