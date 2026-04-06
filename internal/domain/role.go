package domain

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type RoleType string

const (
	RoleTypeAdmin RoleType = "ROLE_ADMIN"
	RoleTypeUser  RoleType = "ROLE_USER"
)

type Role struct {
	ID    uuid.UUID `validate:"required,uuid" json:"id"`
	Title string    `validate:"required" json:"title"`
}

func NewRole(id uuid.UUID, name string) (Role, error) {
	r := Role{
		ID:    id,
		Title: name,
	}

	if err := r.Validate(); err != nil {
		return Role{}, err
	}
	return r, nil
}

func (r Role) Validate() error {
	return validate.V.Struct(r)
}
