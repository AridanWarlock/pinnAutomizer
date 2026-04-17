package domainfixtures

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/google/uuid"
)

func NewRefreshToken(mods ...corefixtures.Mod[domain.RefreshToken]) domain.RefreshToken {
	now := time.Now().UTC()

	var token = domain.RefreshToken{
		Hash:      crypt.Sha256(crypt.GenerateSecureToken()),
		UserID:    uuid.New(),
		Jti:       corefixtures.NewJti(),
		Audit:     corefixtures.NewAuditInfo(),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt: now,
	}

	return corefixtures.Fixture(token, mods)
}
