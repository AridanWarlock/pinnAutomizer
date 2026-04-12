package jwtToken

import (
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrSigningToken          = errors.New("signing token error")
	ErrInvalidToken          = errors.New("invalid token")
	ErrTokenUsedBeforeIssued = errors.New("token used before issued")
	ErrGenerateTokenID       = errors.New("generate token id")
)

var signingMethod = jwt.SigningMethodHS256

type Generator struct {
	secret []byte
}

func NewGenerator(cfg Config) *Generator {
	return &Generator{
		secret: []byte(cfg.Secret),
	}
}

func (g *Generator) Generate(userID uuid.UUID) (domain.AccessToken, error) {
	token, _, err := g.generateAndGetClaims(userID)
	return token, err
}

func (g *Generator) GenerateAndGetClaims(userID uuid.UUID) (domain.AccessToken, domain.JwtClaims, error) {
	token, claims, err := g.generateAndGetClaims(userID)
	if err != nil {
		return "", domain.JwtClaims{}, err
	}

	jwtClaims, err := domain.NewJwtClaims(
		claims.Jti,
		userID,
		claims.IssuedAt,
	)

	if err != nil {
		return "", domain.JwtClaims{}, err
	}
	return token, jwtClaims, nil
}
func (g *Generator) generateAndGetClaims(userID uuid.UUID) (domain.AccessToken, Claims, error) {
	issuedAt := time.Now()

	jti, err := domain.NewJti(uuid.New())
	if err != nil {
		return "", Claims{}, ErrGenerateTokenID
	}

	claims := Claims{
		Jti:      jti,
		UserID:   userID,
		IssuedAt: issuedAt,
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	signedString, err := token.SignedString(g.secret)
	if err != nil {
		return "", Claims{}, ErrSigningToken
	}

	accessToken, err := domain.NewAccessToken(signedString)
	return accessToken, claims, err
}

func (g *Generator) GetClaims(token domain.AccessToken) (domain.JwtClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(string(token), &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return g.secret, nil
	})

	if err != nil {
		return domain.JwtClaims{}, fmt.Errorf("parse claims: %w", err)
	}

	return newClaimsFromToken(parsedToken)
}

func newClaimsFromToken(token *jwt.Token) (domain.JwtClaims, error) {
	if !token.Valid {
		return domain.JwtClaims{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return domain.JwtClaims{}, ErrInvalidToken
	}

	jti := claims.Jti
	if err := jti.Validate(); err != nil {
		return domain.JwtClaims{}, ErrInvalidToken
	}

	userID := claims.UserID
	if userID == uuid.Nil {
		return domain.JwtClaims{}, ErrInvalidToken
	}

	issuedAt := claims.IssuedAt
	if issuedAt.After(time.Now()) {
		return domain.JwtClaims{}, ErrTokenUsedBeforeIssued
	}

	return domain.NewJwtClaims(jti, userID, issuedAt)
}
