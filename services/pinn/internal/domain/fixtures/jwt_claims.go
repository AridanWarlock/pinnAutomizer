package fixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
	"github.com/google/uuid"
)

func NewJwtClaims(mods ...mod[domain.JwtClaims]) domain.JwtClaims {
	claims := domain.JwtClaims{
		Jti:      NewJti(),
		UserID:   uuid.New(),
		IssuedAt: time.Now().Add(-time.Minute),
	}

	return fixture(claims, mods)
}
