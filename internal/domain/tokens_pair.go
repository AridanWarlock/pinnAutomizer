package domain

import "time"

type TokensPair struct {
	AccessToken  Token
	RefreshToken Token
}

type Token struct {
	Value     string
	ExpiresAt time.Time
}

func NewTokensPair(accessToken, refreshToken Token) *TokensPair {
	return &TokensPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}
