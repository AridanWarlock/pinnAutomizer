package domain

import "github.com/google/uuid"

type UsersRoles struct {
	UserID uuid.UUID `validate:"required,uuid"`
	RoleID uuid.UUID `validate:"required,uuid"`
}
