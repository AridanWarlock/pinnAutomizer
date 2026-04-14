package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/middleware"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/response"
)

var (
	ErrTokenIsExpired      = errors.New("token is expired")
	ErrBearerTokenIsNotSet = errors.New("bearer token is not set")
)

type TokenParser interface {
	GetClaims(token core.AccessToken) (domain.JwtClaims, error)
}

type Redis interface {
	Get(ctx context.Context, key string, target any) error
}

type Auth struct {
	redis  Redis
	parser TokenParser

	publicPaths    map[string]struct{}
	publicPrefixes []string
}

func New(
	redis Redis,
	parser TokenParser,
) *Auth {
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

	return &Auth{
		redis:  redis,
		parser: parser,

		publicPaths:    publicPaths,
		publicPrefixes: publicPrefixes,
	}
}

func (a *Auth) Middleware() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if a.isPublicURL(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			log := logger.FromContext(r.Context())

			r, err := a.authenticate(r)

			if err != nil {
				rh := response.NewHandler(w, log)
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

func (a *Auth) isPublicURL(url string) bool {
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

func (a *Auth) authenticate(r *http.Request) (*http.Request, error) {
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

	auditInfo := core.AuditInfoFromContext(ctx)
	if auditInfo.Fingerprint != session.Fingerprint {
		return nil, fmt.Errorf(
			"%w: fingerprint from headers and token not equals",
			errs.ErrSessionIsCompromised,
		)
	}

	authInfo, err := domain.NewAuthInfo(
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

func (a *Auth) extractAccessToken(headers http.Header) (core.AccessToken, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", ErrBearerTokenIsNotSet
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	return core.NewAccessToken(token)
}

func (a *Auth) getSessionFromRedis(ctx context.Context, jti domain.Jti) (domain.RedisSession, error) {
	var session domain.RedisSession
	err := a.redis.Get(ctx, jti.ToRedisKey(), &session)

	if err != nil {
		if errors.Is(err, errs.ErrKeyNotFound) {
			return domain.RedisSession{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, ErrTokenIsExpired)
		}
		return domain.RedisSession{}, fmt.Errorf("redis error: %w", err)
	}
	return session, nil
}
