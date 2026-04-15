package fixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
	"github.com/google/uuid"
)

func NewRedisSession(mods ...mod[domain.RedisSession]) domain.RedisSession {
	s := domain.RedisSession{
		UserID:      uuid.New(),
		Roles:       []domain.Role{NewRole()},
		Fingerprint: NewFingerprint(),
		IssuedAt:    time.Now(),
	}

	return fixture(s, mods)
}
