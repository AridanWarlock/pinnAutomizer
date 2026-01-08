package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	UserIDKey = "userID"
)

var (
	ErrBearerTokenIsNotSet = errors.New("bearer token is not set")
	ErrUnsupportedAuthType = errors.New("unsupported auth type")
)

var publicPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
	"/health",
	"/metrics",
	"/docs",
	"/swagger",
}

var publicPrefixes = []string{
	"/api/v1/scripts/from-translate/",
}

type JwtService interface {
	ValidateAccessToken(ctx context.Context, accessToken string) (uuid.UUID, error)
}

type Middleware struct {
	jwtService JwtService

	log zerolog.Logger
}

func NewMiddleware(jwtService JwtService, log zerolog.Logger) *Middleware {
	return &Middleware{
		jwtService: jwtService,

		log: log.With().Str("component", "middleware: auth").Logger(),
	}
}

func (m *Middleware) IsPublicURL(url string) bool {
	for _, path := range publicPaths {
		if path == url {
			return true
		}
	}

	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}

	return false
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := m.log.With().
			Ctx(r.Context()).
			Str("path", r.URL.Path).
			Str("remote addr", r.RemoteAddr).
			Logger()

		if m.IsPublicURL(r.URL.Path) {
			log.Info().
				Msg("auth to public url")
			next.ServeHTTP(w, r)
			return
		}

		accessToken, err := extractAccessTokenFromHeaders(r)
		if err != nil {
			log.Info().
				Err(err).
				Msg("token extract error")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		userID, err := m.jwtService.ValidateAccessToken(r.Context(), accessToken)
		if err != nil {
			log.Info().
				Err(err).
				Msg("token validate error")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		log.Info().
			Msg("successful auth")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractAccessTokenFromHeaders(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrBearerTokenIsNotSet
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", ErrUnsupportedAuthType
	}

	token := parts[1]
	return token, nil
}
