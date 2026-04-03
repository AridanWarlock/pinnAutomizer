package user_sessions

import (
	"pinnAutomizer/internal/domain"
	"time"

	"github.com/google/uuid"
)

type UserSessionRaw struct {
	ID          uuid.UUID `db:"id"`
	UserID      uuid.UUID `db:"user_id"`
	TokenSha256 []byte    `db:"token_sha256"`
	CreatedAt   time.Time `db:"created_at"`
	ExpiresAt   time.Time `db:"expires_at"`
	Fingerprint []byte    `db:"fingerprint"`
}

func (r UserSessionRaw) Values() []any {
	return []any{
		r.ID,
		r.UserID,
		r.TokenSha256,
		r.CreatedAt,
		r.ExpiresAt,
		r.Fingerprint,
	}
}

func FromModel(model domain.UserSession) UserSessionRaw {
	return UserSessionRaw{
		ID:          model.ID,
		UserID:      model.UserID,
		TokenSha256: model.TokenSha256,
		CreatedAt:   model.CreatedAt,
		ExpiresAt:   model.ExpiresAt,
		Fingerprint: model.Fingerprint,
	}
}

func ToModel(raw UserSessionRaw) domain.UserSession {
	return domain.UserSession{
		ID:          raw.ID,
		UserID:      raw.UserID,
		TokenSha256: raw.TokenSha256,
		CreatedAt:   raw.CreatedAt,
		ExpiresAt:   raw.ExpiresAt,
		Fingerprint: raw.Fingerprint,
	}
}
