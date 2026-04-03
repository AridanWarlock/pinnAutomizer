package domain

import (
	"pinnAutomizer/pkg/validate"
	"time"

	"github.com/google/uuid"
)

type UserSession struct {
	ID          uuid.UUID `validate:"required,uuid" json:"id"`
	UserID      uuid.UUID `validate:"required,uuid" json:"user_id"`
	TokenSha256 []byte    `validate:"required,len=32" json:"token_sha256"`
	CreatedAt   time.Time `validate:"required" json:"created_at"`
	ExpiresAt   time.Time `validate:"required,gtfield=CreatedAt" json:"expires_at"`
	Fingerprint []byte    `validate:"required,len=32" json:"fingerprint"`
}

func NewUserSession(
	userID uuid.UUID,
	tokenSha256 []byte,
	expiresAt time.Time,
	fingerprint []byte,
) (UserSession, error) {
	us := UserSession{
		ID:          uuid.New(),
		UserID:      userID,
		TokenSha256: tokenSha256,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		Fingerprint: fingerprint,
	}

	if err := validate.V.Struct(us); err != nil {
		return UserSession{}, err
	}

	return us, nil
}
