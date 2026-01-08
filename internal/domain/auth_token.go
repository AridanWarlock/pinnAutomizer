package domain

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type AuthToken struct {
	UserID       uuid.UUID `json:"userId"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
}

var authTokensValidator = validator.New(validator.WithRequiredStructEnabled())

func NewAuthToken(userID uuid.UUID, accessToken, refreshToken string) (*AuthToken, error) {
	at := &AuthToken{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	if err := at.Validate(); err != nil {
		return nil, err
	}
	return at, nil
}

func (a *AuthToken) Validate() error {
	err := authTokensValidator.Struct(a)
	if err != nil {
		return fmt.Errorf("authTokenValidator.Sctuct: %w", err)
	}

	return nil
}
