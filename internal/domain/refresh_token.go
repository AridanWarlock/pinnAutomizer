package domain

import (
	"pinnAutomizer/pkg/validate"
	"time"
)

type RefreshToken struct {
	RandomBase64String string    `validate:"required,len=44" json:"random_base64_string"`
	Sha256             []byte    `validate:"required,len=32" json:"token_hash"`
	CreatedAt          time.Time `validate:"required" json:"created_at"`
	ExpiresAt          time.Time `validate:"required,gtfield=CreatedAt" json:"expires_at"`
}

func NewRefreshToken(
	randomBase64String string,
	sha256 []byte,
	expiresAt time.Time,
) (RefreshToken, error) {
	token := RefreshToken{
		RandomBase64String: randomBase64String,
		Sha256:             sha256,
		ExpiresAt:          expiresAt,
		CreatedAt:          time.Now(),
	}

	if err := validate.V.Struct(token); err != nil {
		return RefreshToken{}, err
	}

	return token, nil
}
