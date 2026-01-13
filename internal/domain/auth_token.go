package domain

import (
	"fmt"
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type AuthToken struct {
	UserID       uuid.UUID
	AccessToken  string
	RefreshToken string
}

func NewAuthToken(userID uuid.UUID, accessToken, refreshToken string) (AuthToken, error) {
	at := AuthToken{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	if err := at.Validate(); err != nil {
		return AuthToken{}, err
	}
	return at, nil
}

func (a AuthToken) Validate() error {
	err := validate.V.Struct(a)
	if err != nil {
		return fmt.Errorf("authToken.Validate: %w", err)
	}

	return nil
}
