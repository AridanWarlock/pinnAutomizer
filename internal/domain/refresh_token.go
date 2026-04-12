package domain

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/google/uuid"
)

type RefreshToken struct {
	Hash string `json:"hash" validate:"required,len=64"`

	UserID uuid.UUID `json:"user_id" validate:"required"`
	Jti    Jti       `json:"jti"`

	Fingerprint Fingerprint `json:"fingerprint"`
	UserAgent   UserAgent   `json:"user_agent"`
	IP          UserIP      `json:"ip"`

	ExpiresAt time.Time `json:"expires_at" validate:"required,gtfield=CreatedAt"`
	CreatedAt time.Time `json:"created_at" validate:"required,lte"`
}

func NewRefreshToken(
	hash string,
	userID uuid.UUID,
	jti Jti,
	fingerprint Fingerprint,
	userAgent UserAgent,
	ip UserIP,
	ttl time.Duration,
) (RefreshToken, error) {
	now := time.Now()

	t := RefreshToken{
		Hash:        hash,
		UserID:      userID,
		Jti:         jti,
		Fingerprint: fingerprint,
		UserAgent:   userAgent,
		IP:          ip,
		ExpiresAt:   now.Add(ttl),
		CreatedAt:   now,
	}

	if err := t.Validate(); err != nil {
		return RefreshToken{}, err
	}

	return t, nil
}

func (t *RefreshToken) Validate() error {
	return validate.V.Struct(t)
}
