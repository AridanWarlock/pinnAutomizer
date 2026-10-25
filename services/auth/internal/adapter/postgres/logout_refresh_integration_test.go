package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authLogout "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/logout"
	authRefresh "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/refresh"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func logoutContext(
	t *testing.T,
	fixture refreshIntegration,
	jti core.Jti,
) context.Context {
	t.Helper()
	roles, err := fixture.postgres.GetRolesByUserID(t.Context(), fixture.token.UserID)
	require.NoError(t, err)
	auth, err := core.NewAuthInfo(jti, fixture.token.UserID, roles, time.Now())
	require.NoError(t, err)
	return auth.WithContext(fixture.token.Audit.WithContext(t.Context()))
}

func TestLogoutAuthorizedBeforeRefreshRevokesRotatedSessionIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	// The gateway has authorized logout, but refresh reaches SQL first.
	ctx := logoutContext(t, fixture, fixture.token.Jti)
	refreshed, err := authRefresh.New(fixture.postgres, fixture.redis, fixture.generator).Refresh(ctx, fixture.input)
	require.NoError(t, err)
	claims, err := fixture.generator.GetClaims(refreshed.AccessToken)
	require.NoError(t, err)
	logout := authLogout.New(fixture.postgres, fixture.redis)
	require.NoError(t, logout.Logout(ctx))
	var session core.RedisSession
	require.ErrorIs(t, fixture.redis.Get(ctx, claims.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound,
		"successful logout must revoke the access token issued by the competing refresh")
	require.ErrorIs(t, fixture.redis.Get(ctx, fixture.token.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound)
	_, err = fixture.postgres.GetRefreshTokenByHash(ctx, crypt.Sha256(refreshed.RefreshTokenString))
	require.ErrorIs(t, err, errs.ErrNotFound)
	require.NoError(t, logout.Logout(ctx), "repeated business operation is harmless")
}

type logoutRedis struct {
	*trackedRefreshRedis
	delete func(context.Context, string) error
}

func (r *logoutRedis) Delete(ctx context.Context, key string) error {
	return r.delete(ctx, key)
}

type logoutCompetingRefresh struct {
	*Repository
	reached chan<- struct{}
}

func (p *logoutCompetingRefresh) RotateRefreshToken(
	ctx context.Context,
	oldHash string,
	newHash string,
	jti core.Jti,
) (core.Changed, error) {
	p.reached <- struct{}{}
	return p.Repository.RotateRefreshToken(ctx, oldHash, newHash, jti)
}

func TestLogoutBeforeRefreshRotationDoesNotIssueTokensIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	ctx, cancel := context.WithTimeout(logoutContext(t, fixture, fixture.token.Jti), 10*time.Second)
	defer cancel()
	deleting, rotate, release := make(chan struct{}, 1), make(chan struct{}, 1), make(chan struct{})
	var once sync.Once
	unlock := func() { once.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer func() { cancel(); unlock(); workers.Wait() }()
	redis := &logoutRedis{
		trackedRefreshRedis: fixture.redis,
		delete: func(ctx context.Context, key string) error {
			deleting <- struct{}{}
			select {
			case <-release:
				return fixture.redis.Delete(ctx, key)
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
	logoutDone := make(chan error, 1)
	workers.Go(func() { logoutDone <- authLogout.New(fixture.postgres, redis).Logout(ctx) })
	select {
	case <-deleting:
	case <-ctx.Done():
		t.Fatal("logout did not reach Redis deletion:", ctx.Err())
	}
	refreshDone := make(chan error, 1)
	workers.Go(func() {
		postgres := &logoutCompetingRefresh{Repository: fixture.postgres, reached: rotate}
		out, err := authRefresh.New(postgres, fixture.redis, fixture.generator).Refresh(ctx, fixture.input)
		if err == nil || out != (authRefresh.Output{}) {
			refreshDone <- fmt.Errorf("expected rejected refresh with empty output, got output=%+v error=%v", out, err)
			return
		}
		refreshDone <- err
	})
	select {
	case <-rotate:
	case <-ctx.Done():
		t.Fatal("refresh did not reach rotation:", ctx.Err())
	}
	unlock()
	require.NoError(t, <-logoutDone)
	require.ErrorIs(t, <-refreshDone, errs.ErrAuthorizationFailed)
	workers.Wait()
	for key := range fixture.redis.keys {
		var session core.RedisSession
		require.ErrorIs(t, fixture.redis.Get(ctx, key, &session), errs.ErrKeyNotFound, "key=%s", key)
	}
	_, err := fixture.postgres.GetRefreshTokenByHash(ctx, fixture.token.Hash)
	require.ErrorIs(t, err, errs.ErrNotFound)
}

func TestFailedLogoutKeepsRefreshRecordForRecoveryIntegration(t *testing.T) {
	for _, phase := range []string{"DEL rejected", "DEL response lost", "request cancelled"} {
		t.Run(phase, func(t *testing.T) {
			fixture := newRefreshIntegration(t)
			ctx, cancel := context.WithCancel(logoutContext(t, fixture, fixture.token.Jti))
			defer cancel()
			failure := errors.New("Redis DEL failed")
			redis := &logoutRedis{
				trackedRefreshRedis: fixture.redis,
				delete: func(ctx context.Context, key string) error {
					if phase == "DEL response lost" {
						if err := fixture.redis.Delete(ctx, key); err != nil {
							return err
						}
					}
					if phase == "request cancelled" {
						cancel()
						return fmt.Errorf("%w: %w", errs.ErrServiceUnavailable, ctx.Err())
					}
					return fmt.Errorf("%w: %w", errs.ErrServiceUnavailable, failure)
				},
			}
			err := authLogout.New(fixture.postgres, redis).Logout(ctx)
			require.ErrorIs(t, err, errs.ErrServiceUnavailable)
			if phase == "request cancelled" {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorIs(t, err, failure)
			}
			retryCtx := fixture.token.Audit.WithContext(t.Context())
			persisted, err := fixture.postgres.GetRefreshTokenByHash(retryCtx, fixture.token.Hash)
			require.NoError(t, err, "rollback must retain the current JTI")
			require.Equal(t, fixture.token.Jti, persisted.Jti)
			// DEL may already have revoked access. The known refresh secret allows recovery.
			out, err := authRefresh.New(fixture.postgres, fixture.redis, fixture.generator).Refresh(retryCtx, fixture.input)
			require.NoError(t, err)
			claims, err := fixture.generator.GetClaims(out.AccessToken)
			require.NoError(t, err)
			freshCtx := logoutContext(t, fixture, claims.Jti)
			require.NoError(t, authLogout.New(fixture.postgres, fixture.redis).Logout(freshCtx))
			var session core.RedisSession
			require.ErrorIs(t, fixture.redis.Get(freshCtx, claims.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound)
			_, err = fixture.postgres.GetRefreshTokenByHash(freshCtx, crypt.Sha256(out.RefreshTokenString))
			require.ErrorIs(t, err, errs.ErrNotFound)
		})
	}
}

func TestLogoutWithoutRefreshRecordStillRevokesRequestSessionIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	ctx := logoutContext(t, fixture, fixture.token.Jti)
	require.NoError(t, fixture.postgres.Logout(ctx, fixture.token.UserID, fixture.token.Audit.Fingerprint))
	require.NoError(t, authLogout.New(fixture.postgres, fixture.redis).Logout(ctx))
	var session core.RedisSession
	require.ErrorIs(t, fixture.redis.Get(ctx, fixture.token.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound)
}
