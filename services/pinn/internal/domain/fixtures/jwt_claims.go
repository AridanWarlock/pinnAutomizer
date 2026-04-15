package fixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewJwtClaims(mods ...mod[core.JwtClaims]) core.JwtClaims {
	claims := core.JwtClaims{
		Jti:      NewJti(),
		UserID:   uuid.New(),
		IssuedAt: time.Now().Add(-time.Minute),
	}

	return fixture(claims, mods)
}
