package core

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/google/uuid"
)

type JwtClaims struct {
	Jti      Jti       `json:"jti"`
	UserID   uuid.UUID `json:"user_id" validate:"required,uuid"`
	IssuedAt time.Time `json:"issued_at" validate:"required,lte"`
}

func NewJwtClaims(
	jti Jti,
	userID uuid.UUID,
	issuedAt time.Time,
) (JwtClaims, error) {
	c := JwtClaims{
		Jti:      jti,
		UserID:   userID,
		IssuedAt: issuedAt,
	}

	if err := c.Validate(); err != nil {
		return JwtClaims{}, err
	}

	return c, nil
}

func (c JwtClaims) Validate() error {
	return validate.V.Struct(c)
}
