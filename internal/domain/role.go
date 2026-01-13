package domain

import (
	"github.com/google/uuid"
	"pinnAutomizer/pkg/validate"
)

type Role struct {
	ID   uuid.UUID `validate:"required,uuid"`
	Name string    `validate:"required"`
}

func NewRole(id uuid.UUID, name string) (Role, error) {
	r := Role{
		ID:   id,
		Name: name,
	}

	if err := r.Validate(); err != nil {
		return Role{}, err
	}
	return r, nil
}

func (r Role) Validate() error {
	return validate.V.Struct(r)
}
