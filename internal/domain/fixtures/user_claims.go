package fixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/google/uuid"
)

func NewUserClaims(mods ...mod[domain.UserClaims]) domain.UserClaims {
	now := time.Now()

	userClaims := domain.UserClaims{
		UserID:      uuid.New(),
		Roles:       []domain.Role{NewRole()},
		Fingerprint: NewFingerprint(),
		IssuedAt:    now,
		ExpiresAt:   now.Add(time.Hour),
	}

	return fixture(userClaims, mods)
}
