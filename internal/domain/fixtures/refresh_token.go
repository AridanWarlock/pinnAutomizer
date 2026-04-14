package fixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/google/uuid"
)

func NewRefreshToken(mods ...mod[domain.RefreshToken]) domain.RefreshToken {
	now := time.Now()

	us := domain.RefreshToken{
		Hash:      crypt.Sha256(crypt.GenerateSecureToken()),
		UserID:    uuid.New(),
		Jti:       NewJti(),
		Audit:     NewAuditInfo(),
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}

	return fixture(us, mods)
}
