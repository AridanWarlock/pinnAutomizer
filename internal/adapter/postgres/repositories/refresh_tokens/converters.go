package refresh_tokens

import (
	"net/netip"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/google/uuid"
)

type RefreshTokenRaw struct {
	Hash string `db:"hash"`

	UserID uuid.UUID  `db:"user_id"`
	Jti    domain.Jti `db:"jti"`

	Fingerprint domain.Fingerprint `db:"fingerprint"`
	Agent       domain.UserAgent   `db:"agent"`
	IP          netip.Addr         `db:"ip"`

	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

func (r RefreshTokenRaw) Values() []any {
	return []any{
		r.Hash,

		r.UserID,
		r.Jti,

		r.Fingerprint,
		r.Agent,
		r.IP,

		r.ExpiresAt,
		r.CreatedAt,
	}
}

func FromModel(model domain.RefreshToken) RefreshTokenRaw {
	return RefreshTokenRaw{
		Hash:        model.Hash,
		UserID:      model.UserID,
		Jti:         model.Jti,
		Fingerprint: model.Fingerprint,
		Agent:       model.UserAgent,
		IP:          netip.Addr(model.IP),
		ExpiresAt:   model.ExpiresAt,
		CreatedAt:   model.CreatedAt,
	}
}

func ToModel(raw RefreshTokenRaw) domain.RefreshToken {
	return domain.RefreshToken{
		Hash:        raw.Hash,
		UserID:      raw.UserID,
		Jti:         raw.Jti,
		Fingerprint: raw.Fingerprint,
		UserAgent:   raw.Agent,
		IP:          domain.UserIP(raw.IP),
		ExpiresAt:   raw.ExpiresAt,
		CreatedAt:   raw.CreatedAt,
	}
}
