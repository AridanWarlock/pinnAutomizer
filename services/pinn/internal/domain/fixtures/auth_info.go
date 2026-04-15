package fixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
	"github.com/google/uuid"
)

func NewAuthInfo(mods ...mod[domain.AuthInfo]) domain.AuthInfo {
	auth := domain.AuthInfo{
		Jti:      NewJti(),
		UserID:   uuid.New(),
		Roles:    []domain.Role{NewRole()},
		IssuedAt: time.Now(),
	}

	return fixture(auth, mods)
}
