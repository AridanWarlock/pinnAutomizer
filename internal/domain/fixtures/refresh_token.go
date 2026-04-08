package fixtures

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

func NewRefreshToken(mods ...mod[domain.RefreshToken]) domain.RefreshToken {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	sum256 := sha256.Sum256([]byte("refresh.token"))
	now := time.Now()

	us := domain.RefreshToken{
		RandomBase64String: base64.RawURLEncoding.EncodeToString(b),
		Sha256:             sum256[:],
		CreatedAt:          now,
		ExpiresAt:          now.Add(time.Hour),
	}

	return fixture(us, mods)
}
