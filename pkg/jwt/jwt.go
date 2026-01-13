package jwt

import (
	"context"
	"errors"
	"pinnAutomizer/internal/domain"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrTokenExpired     = errors.New("token expired")
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidSignature = errors.New("invalid token signature")
)

var signingMethod = jwt.SigningMethodHS256

type Config struct {
	AccessTokenSecret    []byte        `env:"JWT_ACCESS_TOKEN_SECRET" required:"true"`
	RefreshTokenSecret   []byte        `env:"JWT_REFRESH_TOKEN_SECRET" required:"true"`
	AccessTokenDuration  time.Duration `env:"JWT_ACCESS_TOKEN_DURATION" required:"true"`
	RefreshTokenDuration time.Duration `env:"JWT_REFRESH_TOKEN_DURATION" required:"true"`
}

type Postgres interface {
	GetAuthTokensByID(ctx context.Context, userID uuid.UUID) (*domain.AuthToken, error)
}

type Service struct {
	accessTokenSecret    []byte
	refreshTokenSecret   []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration

	postgres Postgres
}

func New(c Config, postgres Postgres) *Service {
	return &Service{
		accessTokenSecret:    c.AccessTokenSecret,
		refreshTokenSecret:   c.RefreshTokenSecret,
		accessTokenDuration:  c.AccessTokenDuration,
		refreshTokenDuration: c.RefreshTokenDuration,

		postgres: postgres,
	}
}

type Claims struct {
	jwt.RegisteredClaims
}

func (s *Service) GenerateTokensPair(userID uuid.UUID) (domain.TokensPair, error) {
	accessToken, err := s.GenerateAccessToken(userID)
	if err != nil {
		return domain.TokensPair{}, err
	}

	refreshToken, err := s.GenerateRefreshToken(userID)
	if err != nil {
		return domain.TokensPair{}, err
	}

	return domain.NewTokensPair(accessToken, refreshToken)
}

func (s *Service) GenerateRefreshToken(userID uuid.UUID) (domain.Token, error) {
	subject := userID.String()

	return generateToken(subject, s.refreshTokenDuration, s.refreshTokenSecret)
}

func (s *Service) GenerateAccessToken(userID uuid.UUID) (domain.Token, error) {
	subject := userID.String()

	return generateToken(subject, s.accessTokenDuration, s.accessTokenSecret)
}

func generateToken(subject string, duration time.Duration, secret []byte) (domain.Token, error) {
	expiresAt := time.Now().Add(duration)
	claims := Claims{
		jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	signedString, err := token.SignedString(secret)
	if err != nil {
		return domain.Token{}, err
	}

	return domain.Token{
		Value:     signedString,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) ValidateRefreshToken(ctx context.Context, refreshToken string) (uuid.UUID, error) {
	user, err := s.getAuthTokensFromValidToken(ctx, refreshToken, s.refreshTokenSecret)
	if err != nil {
		return uuid.UUID{}, err
	}

	if user.RefreshToken != refreshToken {
		return uuid.UUID{}, ErrInvalidToken
	}

	return user.UserID, nil
}

func (s *Service) ValidateAccessToken(ctx context.Context, token string) (uuid.UUID, error) {
	authTokens, err := s.getAuthTokensFromValidToken(ctx, token, s.accessTokenSecret)
	if err != nil {
		return uuid.UUID{}, err
	}

	if authTokens.AccessToken != token {
		return uuid.UUID{}, ErrInvalidToken
	}

	return authTokens.UserID, nil
}

func (s *Service) getAuthTokensFromValidToken(ctx context.Context, tokenString string, secret []byte) (*domain.AuthToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}

		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	userID, err := validateToken(token)
	if err != nil {
		return nil, err
	}

	user, err := s.postgres.GetAuthTokensByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return user, nil
}

func validateToken(token *jwt.Token) (uuid.UUID, error) {
	expiresAt, err := token.Claims.GetExpirationTime()
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}
	if expiresAt.Before(time.Now()) {
		return uuid.UUID{}, ErrTokenExpired
	}

	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}
	return userID, nil
}
