package corefixtures

import (
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func NewAuthInfo(mods ...Mod[core.AuthInfo]) core.AuthInfo {
	auth := core.AuthInfo{
		Jti:      NewJti(),
		UserID:   uuid.New(),
		Roles:    []core.Role{NewRole()},
		IssuedAt: time.Now(),
	}

	return Fixture(auth, mods)
}
