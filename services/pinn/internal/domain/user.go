package domain

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" validate:"required,uuid"`
	Login        string    `json:"login" validate:"required"`
	PasswordHash string    `json:"password_hash" validate:"required"`
}

func NewUser(login string, passwordHash string) (User, error) {
	u := User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: passwordHash,
	}

	if err := u.Validate(); err != nil {
		return User{}, err
	}

	return u, nil
}

func (s *User) Validate() error {
	return validate.V.Struct(s)
}
