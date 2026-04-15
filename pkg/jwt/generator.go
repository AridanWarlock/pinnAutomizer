package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
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

func (g *Generator) Generate(userID uuid.UUID) (core.AccessToken, error) {
	token, _, err := g.generateAndGetClaims(userID)
	return token, err
}

func (g *Generator) GenerateAndGetClaims(userID uuid.UUID) (core.AccessToken, core.JwtClaims, error) {
	token, claims, err := g.generateAndGetClaims(userID)
	if err != nil {
		return "", core.JwtClaims{}, err
	}

	jwtClaims, err := core.NewJwtClaims(
		claims.Jti,
		userID,
		claims.IssuedAt,
	)

	if err != nil {
		return "", core.JwtClaims{}, err
	}
	return token, jwtClaims, nil
}
func (g *Generator) generateAndGetClaims(userID uuid.UUID) (core.AccessToken, Claims, error) {
	issuedAt := time.Now()

	jti, err := core.NewJti(uuid.New())
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

	accessToken, err := core.NewAccessToken(signedString)
	return accessToken, claims, err
}

func (g *Generator) GetClaims(token core.AccessToken) (core.JwtClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(string(token), &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return g.secret, nil
	})

	if err != nil {
		return core.JwtClaims{}, fmt.Errorf("parse claims: %w", err)
	}

	return newClaimsFromToken(parsedToken)
}

func newClaimsFromToken(token *jwt.Token) (core.JwtClaims, error) {
	if !token.Valid {
		return core.JwtClaims{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return core.JwtClaims{}, ErrInvalidToken
	}

	jti := claims.Jti
	if err := jti.Validate(); err != nil {
		return core.JwtClaims{}, ErrInvalidToken
	}

	userID := claims.UserID
	if userID == uuid.Nil {
		return core.JwtClaims{}, ErrInvalidToken
	}

	issuedAt := claims.IssuedAt
	if issuedAt.After(time.Now()) {
		return core.JwtClaims{}, ErrTokenUsedBeforeIssued
	}

	return core.NewJwtClaims(jti, userID, issuedAt)
}
