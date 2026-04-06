package refresh_token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

var DefaultEncoder = base64.RawURLEncoding

type Config struct {
	Ttl time.Duration `env:"TOKEN_TTL,required"`
}

type Generator struct {
	ttl     time.Duration
	encoder *base64.Encoding
}

func New(c Config) *Generator {
	return &Generator{
		ttl:     c.Ttl,
		encoder: DefaultEncoder,
	}
}

func (g *Generator) Generate() (domain.RefreshToken, error) {
	randomBase64String := g.generateRandomToken()
	sha256Bytes := sha256.Sum256([]byte(randomBase64String))
	expiresAt := time.Now().Add(g.ttl)

	return domain.NewRefreshToken(
		randomBase64String,
		sha256Bytes[:],
		expiresAt,
	)
}

func (g *Generator) generateRandomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)

	return g.encoder.EncodeToString(b)
}
