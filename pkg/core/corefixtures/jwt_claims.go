package corefixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewJwtClaims(mods ...Mod[core.JwtClaims]) core.JwtClaims {
	claims := core.JwtClaims{
		Jti:      NewJti(),
		UserID:   uuid.New(),
		IssuedAt: time.Now().Add(-time.Minute),
	}

	return Fixture(claims, mods)
}
