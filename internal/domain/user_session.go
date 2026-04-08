package domain

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type UserSession struct {
	ID          uuid.UUID   `validate:"required,uuid" json:"id"`
	UserID      uuid.UUID   `validate:"required,uuid" json:"user_id"`
	TokenSha256 []byte      `validate:"required,len=32" json:"token_sha256"`
	CreatedAt   time.Time   `validate:"required" json:"created_at"`
	ExpiresAt   time.Time   `validate:"required,gtfield=CreatedAt" json:"expires_at"`
	Fingerprint Fingerprint `json:"fingerprint"`
}

func NewUserSession(
	userID uuid.UUID,
	tokenSha256 []byte,
	expiresAt time.Time,
	fingerprint Fingerprint,
	now time.Time,
) (UserSession, error) {
	us := UserSession{
		ID:          uuid.New(),
		UserID:      userID,
		TokenSha256: tokenSha256,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		Fingerprint: fingerprint,
	}

	if err := us.Validate(); err != nil {
		return UserSession{}, err
	}

	return us, nil
}

func (s UserSession) Validate() error {
	if err := validate.V.Struct(s); err != nil {
		return err
	}

	if err := s.Fingerprint.Validate(); err != nil {
		return err
	}

	return nil
}
