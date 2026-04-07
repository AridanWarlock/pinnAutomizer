package jwtToken

import (
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const Issuer = "API-Gateway"

var (
	ErrInvalidToken = errors.New("invalid token")
)

var signingMethod = jwt.SigningMethodHS256

type Claims struct {
	UserID uuid.UUID     `json:"user_id"`
	Roles  []domain.Role `json:"roles"`

	jwt.RegisteredClaims
}

type Generator struct {
	secret []byte
	ttl    time.Duration
}

func NewGenerator(c Config) *Generator {
	return &Generator{
		secret: []byte(c.Secret),
		ttl:    c.TokenTTL,
	}
}

func (g *Generator) Generate(user domain.User, roles []domain.Role) (domain.AccessToken, error) {
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(g.ttl)

	claims := Claims{
		UserID: user.ID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			Issuer:    Issuer,
		},
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	signedString, err := token.SignedString(g.secret)
	if err != nil {
		return "", err
	}

	return domain.AccessToken(signedString), nil
}

func (g *Generator) GetClaims(token domain.AccessToken) (domain.UserClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(string(token), &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}

		return g.secret, nil
	})

	if err != nil {
		return domain.UserClaims{}, err
	}

	return validateToken(parsedToken)
}

func validateToken(token *jwt.Token) (domain.UserClaims, error) {
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return domain.UserClaims{}, ErrInvalidToken
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil {
		return domain.UserClaims{}, jwt.ErrInvalidType
	}
	if expiresAt.Before(time.Now()) {
		return domain.UserClaims{}, jwt.ErrTokenExpired
	}

	issuedAt, err := claims.GetIssuedAt()
	if err != nil {
		return domain.UserClaims{}, jwt.ErrInvalidType
	}
	if issuedAt.After(time.Now()) {
		return domain.UserClaims{}, jwt.ErrTokenUsedBeforeIssued
	}

	issuer, err := claims.GetIssuer()
	if err != nil {
		return domain.UserClaims{}, jwt.ErrInvalidType
	}
	if issuer != Issuer {
		return domain.UserClaims{}, jwt.ErrTokenInvalidIssuer
	}

	return domain.NewUserClaims(
		claims.UserID,
		claims.Roles,
		issuedAt.Time,
		expiresAt.Time,
	)
}
