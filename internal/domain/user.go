package domain

import (
	"fmt"
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
}

func NewUser(login string, passwordHash string) (*User, error) {
	id := uuid.New()

	u := &User{
		ID:           id,
		Login:        login,
		PasswordHash: passwordHash,
	}

	if err := u.Validate(); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *User) Validate() error {
	err := validate.V.Struct(s)
	if err != nil {
		return fmt.Errorf("user.Validate: %w", err)
	}

	return nil
}
