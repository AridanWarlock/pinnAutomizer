package corefixtures

import (
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func NewJwtClaims(mods ...Mod[core.JwtClaims]) core.JwtClaims {
	claims := core.JwtClaims{
		Jti:      NewJti(),
		UserID:   uuid.New(),
		IssuedAt: time.Now().Add(-time.Minute),
	}

	return Fixture(claims, mods)
}
