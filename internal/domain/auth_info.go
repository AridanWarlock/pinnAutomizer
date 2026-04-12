package domain

import (
	"context"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/google/uuid"
)

type authInfoKey struct{}

type AuthInfo struct {
	Jti      Jti       `json:"jti"`
	UserID   uuid.UUID `json:"user_id" validate:"required,uuid"`
	Roles    []Role    `json:"roles" validate:"required"`
	IssuedAt time.Time `json:"issued_at" validate:"required,lte"`
}

func NewAuthInfo(
	jti Jti,
	userID uuid.UUID,
	roles []Role,
	issuedAt time.Time,
) (AuthInfo, error) {
	a := AuthInfo{
		Jti:      jti,
		UserID:   userID,
		Roles:    roles,
		IssuedAt: issuedAt,
	}

	if err := a.Validate(); err != nil {
		return AuthInfo{}, ErrInvalidAuthInfo
	}
	return a, nil
}

func (a AuthInfo) Validate() error {
	return validate.V.Struct(a)
}

func (a AuthInfo) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, authInfoKey{}, a)
}

func AuthInfoFromContext(ctx context.Context) AuthInfo {
	v, ok := ctx.Value(authInfoKey{}).(AuthInfo)
	if !ok {
		panic("no auth info in context")
	}
	return v
}
