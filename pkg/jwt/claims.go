package jwt

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Jti      core.Jti  `json:"jti"`
	UserID   uuid.UUID `json:"sub"`
	IssuedAt time.Time `json:"iat"`
}

func (c Claims) GetSubject() (string, error) {
	return c.UserID.String(), nil
}

func (c Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(c.IssuedAt), nil
}

func (c Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	return nil, nil
}

func (c Claims) GetIssuer() (string, error) {
	return "", nil
}

func (c Claims) GetNotBefore() (*jwt.NumericDate, error) {
	return nil, nil
}

func (c Claims) GetAudience() (jwt.ClaimStrings, error) {
	return nil, nil
}
