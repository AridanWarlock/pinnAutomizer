package domain

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"passwordHash"`
}

var usersValidator = validator.New(validator.WithRequiredStructEnabled())

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
	err := usersValidator.Struct(s)
	if err != nil {
		return fmt.Errorf("usersValidator.Sctuct: %w", err)
	}

	return nil
}
