package domain

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type UserClaims struct {
	UserID      uuid.UUID   `validate:"required,uuid" json:"user_id"`
	Roles       []Role      `validate:"required" json:"roles"`
	Fingerprint Fingerprint `validate:"required,len=32" json:"fingerprint"`
	IssuedAt    time.Time   `validate:"required" json:"issued_at"`
	ExpiresAt   time.Time   `validate:"required" json:"expires_at"`
}

func NewUserClaims(
	userID uuid.UUID,
	roles []Role,
	fingerprint Fingerprint,
	issuedAt time.Time,
	expiresAt time.Time,
) (UserClaims, error) {
	uc := UserClaims{
		UserID:      userID,
		Roles:       roles,
		Fingerprint: fingerprint,
		IssuedAt:    issuedAt,
		ExpiresAt:   expiresAt,
	}

	if err := uc.Validate(); err != nil {
		return UserClaims{}, err
	}

	return uc, nil
}

func (c UserClaims) Validate() error {
	if err := c.Fingerprint.Validate(); err != nil {
		return err
	}

	if err := validate.V.Struct(c); err != nil {
		return err
	}

	return nil
}
