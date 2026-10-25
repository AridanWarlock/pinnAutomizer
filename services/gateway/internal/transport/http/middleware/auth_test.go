package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jwt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

type authFixture struct {
	token   core.AccessToken
	claims  core.JwtClaims
	session core.RedisSession
	audit   core.AuditInfo
	parser  *jwt.Generator
}

func newAuthFixture(t *testing.T) authFixture {
	t.Helper()
	userID := uuid.New()
	role, err := core.NewRole(uuid.New(), string(core.RoleTypeUser))
	require.NoError(t, err)
	fingerprint, err := core.NewFingerprint(strings.Repeat("a", 64))
	require.NoError(t, err)
	ip, err := core.NewUserIP("127.0.0.1")
	require.NoError(t, err)
	agent, err := core.NewUserAgent("auth-test")
	require.NoError(t, err)
	session, err := core.NewRedisSession(userID, []core.Role{role}, fingerprint, time.Now())
	require.NoError(t, err)
	parser := jwt.NewGenerator(jwt.Config{Secret: "auth-tests-only-secret"})
	token, claims, err := parser.GenerateAndGetClaims(userID)
	require.NoError(t, err)
	return authFixture{
		token:   token,
		claims:  claims,
		session: session,
		audit:   core.NewAuditInfo(fingerprint, ip, agent),
		parser:  parser,
	}
}

func authHandler(redis Redis, fixture authFixture) http.Handler {
	return Auth(redis, fixture.parser, &MockKafkaWriter{}, "auth.session.compromised")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := core.MustAuthInfoFromContext(r.Context())
		w.Header().Set("X-Test-User", auth.UserID.String())
		w.WriteHeader(http.StatusNoContent)
	}))
}

func authRequest(t *testing.T, handler http.Handler, fixture authFixture) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	request.Header.Set("Authorization", "Bearer "+string(fixture.token))
	ctx := logger.WithContext(t.Context(), zerolog.Nop())
	request = request.WithContext(fixture.audit.WithContext(ctx))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestAuthorizationRechecksRedisAndDistinguishesMissingSessionFromFailure(t *testing.T) {
	t.Parallel()
	fixture := newAuthFixture(t)
	calls := 0
	redis := &MockRedis{
		GetFunc: func(_ context.Context, key string, target any) error {
			require.Equal(t, fixture.claims.Jti.ToRedisKey(), key)
			calls++
			switch calls {
			case 1:
				*target.(*core.RedisSession) = fixture.session
				return nil
			case 2:
				return errs.ErrKeyNotFound
			default:
				return errors.New("redis connection refused")
			}
		},
	}
	handler := authHandler(redis, fixture)
	response := authRequest(t, handler, fixture)
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, fixture.session.UserID.String(), response.Header().Get("X-Test-User"))
	response = authRequest(t, handler, fixture)
	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Empty(t, response.Header().Get("X-Test-User"))
	response = authRequest(t, handler, fixture)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Empty(t, response.Header().Get("X-Test-User"))
	require.Equal(t, 3, calls)
}

func TestPublicAndInvalidRequestsDoNotReadSessionStore(t *testing.T) {
	t.Parallel()
	fixture := newAuthFixture(t)
	for _, scenario := range []struct {
		name   string
		path   string
		token  string
		status int
	}{
		{name: "health is public", path: "/health", status: http.StatusNoContent},
		{name: "missing token", path: "/api/v1/tasks", status: http.StatusUnauthorized},
		{name: "invalid token", path: "/api/v1/tasks", token: "invalid", status: http.StatusUnauthorized},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			redis := &MockRedis{}
			handler := Auth(redis, fixture.parser, &MockKafkaWriter{}, "auth.session.compromised")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodGet, scenario.path, nil)
			if scenario.token != "" {
				request.Header.Set("Authorization", "Bearer "+scenario.token)
			}
			request = request.WithContext(logger.WithContext(t.Context(), zerolog.Nop()))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, scenario.status, response.Code)
			require.Empty(t, redis.GetCalls())
		})
	}
}
