package fixtures

import (
	"crypto/sha256"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/google/uuid"
)

func NewUserSession(mods ...mod[domain.UserSession]) domain.UserSession {
	tokenSha256 := sha256.Sum256([]byte("token"))
	fingerprint := sha256.Sum256([]byte("fingerprint"))
	now := time.Now()

	us := domain.UserSession{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		TokenSha256: tokenSha256[:],
		CreatedAt:   now,
		ExpiresAt:   now.Add(time.Hour),
		Fingerprint: fingerprint[:],
	}

	return fixture(us, mods)
}
