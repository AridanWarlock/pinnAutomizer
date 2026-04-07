package http_middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
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
}

var publicPrefixes = []string{
	"/swagger/",
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

			rh := http_response.NewHandler(w, log)
			accessToken, err := extractAccessTokenFromHeaders(r)
			if err != nil {
				rh.ErrorResponse(
					fmt.Errorf("%w: extract access token from headers: %v", errs.ErrAuthorizationFailed, err),
					"failed to extract access token from headers",
				)
				return
			}

			claims, err := tokenParser.GetClaims(domain.AccessToken(accessToken))
			if err != nil {
				rh.ErrorResponse(
					fmt.Errorf("%w: parse user claims from access token: %v", errs.ErrAuthorizationFailed, err),
					"failed to parse valid claims from access token",
				)
				return
			}

			ctx = context.WithValue(ctx, UserClaimsKey, claims)
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
