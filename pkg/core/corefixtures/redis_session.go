package corefixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewRedisSession(mods ...Mod[core.RedisSession]) core.RedisSession {
	s := core.RedisSession{
		UserID:      uuid.New(),
		Roles:       []core.Role{NewRole()},
		Fingerprint: NewFingerprint(),
		IssuedAt:    time.Now(),
	}

	return Fixture(s, mods)
}
