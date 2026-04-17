package corefixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
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
