package postgres

import (
	"net/netip"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

type RefreshTokenRaw struct {
	Hash string `db:"hash"`

	UserID uuid.UUID `db:"user_id"`
	Jti    core.Jti  `db:"jti"`

	Fingerprint core.Fingerprint `db:"fingerprint"`
	Agent       core.UserAgent   `db:"agent"`
	IP          netip.Addr       `db:"ip"`

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

func fromRefreshTokenModel(model domain.RefreshToken) RefreshTokenRaw {
	return RefreshTokenRaw{
		Hash:        model.Hash,
		UserID:      model.UserID,
		Jti:         model.Jti,
		Fingerprint: model.Audit.Fingerprint,
		Agent:       model.Audit.Agent,
		IP:          netip.Addr(model.Audit.IP),
		ExpiresAt:   model.ExpiresAt,
		CreatedAt:   model.CreatedAt,
	}
}

func toRefreshTokenModel(raw RefreshTokenRaw) domain.RefreshToken {
	return domain.RefreshToken{
		Hash:   raw.Hash,
		UserID: raw.UserID,
		Jti:    raw.Jti,
		Audit: core.NewAuditInfo(
			raw.Fingerprint,
			core.UserIP(raw.IP),
			raw.Agent,
		),
		ExpiresAt: raw.ExpiresAt,
		CreatedAt: raw.CreatedAt,
	}
}
