package core_http_middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

const UserClaimsKey = "userClaimsKey"

var (
	ErrBearerTokenIsNotSet = errors.New("bearer token is not set")
	ErrUnsupportedAuthType = errors.New("unsupported auth type")
)

var publicPaths = map[string]struct{}{
	"/api/v1/auth/login":    {},
	"/api/v1/auth/register": {},
	"/api/v1/auth/refresh":  {},
	"/health":               {},
	"/metrics":              {},
	"/docs":                 {},
	"/swagger":              {},
}

var publicPrefixes = []string{
	"/api/v1/scripts/from-translate/",
}

type TokenParser interface {
	GetClaims(token domain.AccessToken) (domain.UserClaims, error)
}

func Authentication(tokenParser TokenParser) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := logger.FromContext(ctx)

			if isPublicURL(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			accessToken, err := extractAccessTokenFromHeaders(r)
			if err != nil {
				log.Info().Err(err).Msg("extract access token from headers")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := tokenParser.GetClaims(domain.AccessToken(accessToken))
			if err != nil {
				log.Info().Err(err).Msg("parsing claims from access token")
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			ctx = context.WithValue(r.Context(), UserClaimsKey, claims)
			log.Info().Msg("successful auth")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isPublicURL(url string) bool {
	if _, ok := publicPaths[url]; ok {
		return true
	}

	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}

	return false
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
