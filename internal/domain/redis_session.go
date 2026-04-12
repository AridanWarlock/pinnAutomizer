package domain

import (
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/google/uuid"
)

type RedisSession struct {
	UserID      uuid.UUID   `validate:"required,uuid" json:"user_id"`
	Roles       []Role      `validate:"required" json:"roles"`
	Fingerprint Fingerprint `json:"fingerprint"`
	IssuedAt    time.Time   `validate:"required" json:"issued_at"`
}

func NewRedisSession(
	userID uuid.UUID,
	roles []Role,
	fingerprint Fingerprint,
	issuedAt time.Time,
) (RedisSession, error) {
	s := RedisSession{
		UserID:      userID,
		Roles:       roles,
		Fingerprint: fingerprint,
		IssuedAt:    issuedAt,
	}

	if err := s.Validate(); err != nil {
		return RedisSession{}, err
	}
	return s, nil
}

func (s RedisSession) Validate() error {
	return validate.V.Struct(s)
}
