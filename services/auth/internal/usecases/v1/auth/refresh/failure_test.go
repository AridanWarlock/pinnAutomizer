package authRefresh

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jwt"
)

type refreshFixture struct {
	input     Input
	token     domain.RefreshToken
	roles     []core.Role
	generator *jwt.Generator
}

func newRefreshFixture(t *testing.T) refreshFixture {
	t.Helper()
	userID := uuid.New()
	fingerprint, err := core.NewFingerprint(strings.Repeat("a", 64))
	require.NoError(t, err)
	ip, err := core.NewUserIP("127.0.0.1")
	require.NoError(t, err)
	agent, err := core.NewUserAgent("refresh-test")
	require.NoError(t, err)
	audit := core.NewAuditInfo(fingerprint, ip, agent)
	jti, err := core.NewJti(uuid.New())
	require.NoError(t, err)
	secret := crypt.GenerateSecureToken()
	token, err := domain.NewRefreshToken(crypt.Sha256(secret), userID, jti, audit, time.Hour)
	require.NoError(t, err)
	role, err := core.NewRole(uuid.New(), string(core.RoleTypeUser))
	require.NoError(t, err)
	return refreshFixture{
		input:     Input{RefreshTokenString: secret},
		token:     token,
		roles:     []core.Role{role},
		generator: jwt.NewGenerator(jwt.Config{Secret: "refresh-tests-only-secret"}),
	}
}

func TestRedisPreparationFailureKeepsRefreshTokenUsable(t *testing.T) {
	t.Parallel()
	fixture := newRefreshFixture(t)
	failure := errors.New("Redis unavailable")
	consumed := false
	postgres := &MockPostgres{
		GetRefreshTokenByHashFunc: func(context.Context, string) (domain.RefreshToken, error) {
			if consumed {
				return domain.RefreshToken{}, errs.ErrNotFound
			}
			return fixture.token, nil
		},
		GetRolesByUserIDFunc: func(context.Context, uuid.UUID) ([]core.Role, error) {
			return fixture.roles, nil
		},
		RotateRefreshTokenFunc: func(context.Context, string, string, core.Jti) (core.Changed, error) {
			consumed = true
			return true, nil
		},
	}
	redis := &MockRedis{
		DeleteFunc: func(context.Context, string) error {
			return nil
		},
		SetFunc: func(context.Context, string, any, time.Duration) error {
			return fmt.Errorf("%w: %w", errs.ErrServiceUnavailable, failure)
		},
	}
	uc := New(postgres, redis, fixture.generator)
	ctx := fixture.token.Audit.WithContext(t.Context())
	out, err := uc.Refresh(ctx, fixture.input)
	require.ErrorIs(t, err, failure)
	require.Empty(t, out)
	require.False(t, consumed, "failed Redis preparation must not consume the only known refresh token")
	require.Empty(t, redis.DeleteCalls(), "old access session must remain usable on SET failure")
	redis.SetFunc = func(context.Context, string, any, time.Duration) error {
		return nil
	}
	out, err = uc.Refresh(ctx, fixture.input)
	require.NoError(t, err, "same refresh token must still be usable after failed preparation")
	require.NotEmpty(t, out.AccessToken)
	require.NotEmpty(t, out.RefreshTokenString)
	require.True(t, consumed)
}

func TestOldSessionDeletionFailureDoesNotRotateRefreshToken(t *testing.T) {
	t.Parallel()
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "Redis error", true: "request cancelled after SET"}[cancelled], func(t *testing.T) {
			fixture := newRefreshFixture(t)
			postgres, redis := refreshDependencies(fixture)
			failure := errors.New("Redis DEL failed")
			ctx, cancel := context.WithCancel(fixture.token.Audit.WithContext(t.Context()))
			defer cancel()
			if cancelled {
				failure = context.Canceled
				redis.SetFunc = func(context.Context, string, any, time.Duration) error {
					cancel()
					return nil
				}
			}
			redis.DeleteFunc = func(ctx context.Context, key string) error {
				require.Equal(t, fixture.token.Jti.ToRedisKey(), key)
				if cancelled {
					return fmt.Errorf("%w: %w", errs.ErrServiceUnavailable, ctx.Err())
				}
				return fmt.Errorf("%w: %w", errs.ErrServiceUnavailable, failure)
			}
			out, err := New(postgres, redis, fixture.generator).Refresh(ctx, fixture.input)
			require.ErrorIs(t, err, failure)
			require.ErrorIs(t, err, errs.ErrServiceUnavailable)
			require.Empty(t, out)
			require.Empty(t, postgres.RotateRefreshTokenCalls(), "DEL failure must leave PostgreSQL refresh token unchanged")
		})
	}
}

func TestConcurrentLoserPreservesAuthorizationFailureWhenCleanupFails(t *testing.T) {
	t.Parallel()
	fixture := newRefreshFixture(t)
	postgres, redis := refreshDependencies(fixture)
	postgres.RotateRefreshTokenFunc = func(context.Context, string, string, core.Jti) (core.Changed, error) {
		return false, nil
	}
	failure := errors.New("candidate cleanup failed")
	redis.DeleteFunc = func(_ context.Context, key string) error {
		if key == fixture.token.Jti.ToRedisKey() {
			return nil
		}
		return fmt.Errorf("%w: %w", errs.ErrServiceUnavailable, failure)
	}
	out, err := New(postgres, redis, fixture.generator).Refresh(fixture.token.Audit.WithContext(t.Context()), fixture.input)
	require.Empty(t, out)
	require.ErrorIs(t, err, errs.ErrAuthorizationFailed)
	require.ErrorIs(t, err, failure)
	require.Len(t, redis.DeleteCalls(), 2, "delete old session, then discard only this request's candidate")
}
