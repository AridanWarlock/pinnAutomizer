package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpmv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

var (
	ErrTokenIsExpired      = errors.New("token is expired")
	ErrBearerTokenIsNotSet = errors.New("bearer token is not set")
)

type TokenParser interface {
	GetClaims(token core.AccessToken) (core.JwtClaims, error)
}

type Redis interface {
	Get(ctx context.Context, key string, target any) error
}

type auth struct {
	redis  Redis
	parser TokenParser

	publicPaths    map[string]struct{}
	publicPrefixes []string
}

func Auth(
	redis Redis,
	parser TokenParser,
) httpmv.Middleware {
	publicPaths := map[string]struct{}{
		"/api/v1/auth/login":    {},
		"/api/v1/auth/register": {},
		"/api/v1/auth/refresh":  {},
		"/health":               {},
		"/metrics":              {},
		"/docs":                 {},
	}

	publicPrefixes := []string{
		"/swagger/",
		"/api/v1/scripts/from-translate/",
	}

	auth := auth{
		redis:  redis,
		parser: parser,

		publicPaths:    publicPaths,
		publicPrefixes: publicPrefixes,
	}
	return auth.middleware()
}

func (a *auth) middleware() httpmv.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if a.isPublicURL(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			log := logger.FromContext(r.Context())

			r, err := a.authenticate(r)

			if err != nil {
				rh := httpout.NewHandler(w, log)
				rh.ErrorResponse(
					fmt.Errorf("%w: %v", errs.ErrAuthorizationFailed, err),
					"failed to authenticate",
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (a *auth) isPublicURL(url string) bool {
	if _, ok := a.publicPaths[url]; ok {
		return true
	}

	for _, prefix := range a.publicPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}

	return false
}

func (a *auth) authenticate(r *http.Request) (*http.Request, error) {
	ctx := r.Context()

	accessToken, err := a.extractAccessToken(r.Header)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	claims, err := a.parser.GetClaims(accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	jti := claims.Jti

	session, err := a.getSessionFromRedis(ctx, jti)
	if err != nil {
		return nil, fmt.Errorf("get session from redis: %w", err)
	}

	auditInfo := core.MustAuditInfoFromContext(ctx)
	if auditInfo.Fingerprint != session.Fingerprint {
		return nil, fmt.Errorf(
			"%w: fingerprint from headers and token not equals",
			errs.ErrSessionIsCompromised,
		)
	}

	authInfo, err := core.NewAuthInfo(
		jti,
		session.UserID,
		session.Roles,
		session.IssuedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create auth info: %w", err)
	}

	ctx = authInfo.WithContext(ctx)
	return r.WithContext(ctx), nil
}

func (a *auth) extractAccessToken(headers http.Header) (core.AccessToken, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", ErrBearerTokenIsNotSet
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	return core.NewAccessToken(token)
}

func (a *auth) getSessionFromRedis(ctx context.Context, jti core.Jti) (core.RedisSession, error) {
	var session core.RedisSession
	err := a.redis.Get(ctx, jti.ToRedisKey(), &session)

	if err != nil {
		if errors.Is(err, errs.ErrKeyNotFound) {
			return core.RedisSession{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, ErrTokenIsExpired)
		}
		return core.RedisSession{}, fmt.Errorf("redis error: %w", err)
	}
	return session, nil
}
