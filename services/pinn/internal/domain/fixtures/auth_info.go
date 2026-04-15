package fixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewAuthInfo(mods ...mod[core.AuthInfo]) core.AuthInfo {
	auth := core.AuthInfo{
		Jti:      NewJti(),
		UserID:   uuid.New(),
		Roles:    []core.Role{NewRole()},
		IssuedAt: time.Now(),
	}

	return fixture(auth, mods)
}
