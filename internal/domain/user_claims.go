package domain

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type UserClaims struct {
	UserID    uuid.UUID `validate:"required,uuid" json:"user_id"`
	Roles     []Role    `validate:"required" json:"roles"`
	IssuedAt  time.Time `validate:"required" json:"issued_at"`
	ExpiresAt time.Time `validate:"required" json:"expires_at"`
}

func NewUserClaims(
	userID uuid.UUID,
	roles []Role,
	issuedAt time.Time,
	expiresAt time.Time,
) (UserClaims, error) {
	uc := UserClaims{
		UserID:    userID,
		Roles:     roles,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}

	if err := validate.V.Struct(uc); err != nil {
		return UserClaims{}, err
	}

	return uc, nil
}
