package domain

import (
	"fmt"
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"password_hash"`
}

func NewUser(login string, passwordHash string) (User, error) {
	id := uuid.New()

	u := User{
		ID:           id,
		Login:        login,
		PasswordHash: passwordHash,
	}

	if err := u.Validate(); err != nil {
		return User{}, err
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
