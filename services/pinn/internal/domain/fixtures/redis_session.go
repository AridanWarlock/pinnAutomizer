package fixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewRedisSession(mods ...mod[core.RedisSession]) core.RedisSession {
	s := core.RedisSession{
		UserID:      uuid.New(),
		Roles:       []core.Role{NewRole()},
		Fingerprint: NewFingerprint(),
		IssuedAt:    time.Now(),
	}

	return fixture(s, mods)
}
