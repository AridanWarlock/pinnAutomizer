package domain

import "github.com/google/uuid"

type UsersRoles struct {
	UserID uuid.UUID `validate:"required,uuid" json:"user_id"`
	RoleID uuid.UUID `validate:"required,uuid" json:"role_id"`
}
