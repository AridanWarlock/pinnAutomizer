package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/cleanupSessions"
	authLogout "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/logout"
	authRefresh "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/refresh"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
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
	postgres := &pausedLogoutPostgres{
		Repository: fixture.postgres,
		locked:     deleting,
		release:    release,
	}
	logoutDone := make(chan error, 1)
	workers.Go(func() { logoutDone <- authLogout.New(postgres, fixture.redis).Logout(ctx) })
	select {
	case <-deleting:
	case <-ctx.Done():
		t.Fatal("logout did not lock the refresh row:", ctx.Err())
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

func TestFailedRedisCleanupCannotRestoreRefreshIntegration(t *testing.T) {
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
			_, err = fixture.postgres.GetRefreshTokenByHash(retryCtx, fixture.token.Hash)
			require.ErrorIs(t, err, errs.ErrNotFound, "committed logout must consume refresh before DEL")
			out, err := authRefresh.New(fixture.postgres, fixture.redis, fixture.generator).Refresh(retryCtx, fixture.input)
			require.ErrorIs(t, err, errs.ErrAuthorizationFailed)
			require.Empty(t, out)
			pending, err := fixture.postgres.GetSessionRevocations(retryCtx, fixture.token.UserID, fixture.token.Audit.Fingerprint)
			require.NoError(t, err)
			require.Contains(t, pending, fixture.token.Jti)
			// A fresh cleanup instance has no request state; the retry is driven by PostgreSQL.
			_, err = cleanupSessions.New(fixture.postgres, fixture.redis).Cleanup(retryCtx, cleanupSessions.Input{})
			require.NoError(t, err)
			var session core.RedisSession
			require.ErrorIs(t, fixture.redis.Get(retryCtx, fixture.token.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound)
			pending, err = fixture.postgres.GetSessionRevocations(retryCtx, fixture.token.UserID, fixture.token.Audit.Fingerprint)
			require.NoError(t, err)
			require.Empty(t, pending)

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

type pausedLogoutPostgres struct {
	*Repository
	locked  chan<- struct{}
	release <-chan struct{}
}

func (p *pausedLogoutPostgres) GetJtiByFingerprintForUpdate(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
) (core.Jti, error) {
	jti, err := p.Repository.GetJtiByFingerprintForUpdate(ctx, userID, fingerprint)
	if err != nil {
		return core.Jti{}, err
	}
	p.locked <- struct{}{}
	select {
	case <-p.release:
		return jti, nil
	case <-ctx.Done():
		return core.Jti{}, ctx.Err()
	}
}

type interruptedLogoutPostgres struct {
	*Repository
	failure     error
	afterCommit bool
}

func (p *interruptedLogoutPostgres) Logout(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
) error {
	if !p.afterCommit {
		return p.failure
	}
	return p.Repository.Logout(ctx, userID, fingerprint)
}

func (p *interruptedLogoutPostgres) InTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	if err := p.Repository.InTransaction(ctx, fn); err != nil {
		return err
	}
	return p.failure
}

func TestLogoutTransactionFailureBeforeAndAfterCommitIntegration(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback before commit", true: "commit response lost"}[committed], func(t *testing.T) {
			fixture := newRefreshIntegration(t)
			ctx := logoutContext(t, fixture, fixture.token.Jti)
			failure := errors.New("SQL failure")
			postgres := &interruptedLogoutPostgres{
				Repository:  fixture.postgres,
				failure:     failure,
				afterCommit: committed,
			}
			require.ErrorIs(t, authLogout.New(postgres, fixture.redis).Logout(ctx), failure)
			// No Redis effect before confirmed commit. A committed but unknown outcome is recoverable.
			var session core.RedisSession
			require.NoError(t, fixture.redis.Get(ctx, fixture.token.Jti.ToRedisKey(), &session))
			pending, err := fixture.postgres.GetSessionRevocations(ctx, fixture.token.UserID, fixture.token.Audit.Fingerprint)
			require.NoError(t, err)
			_, err = fixture.postgres.GetRefreshTokenByHash(ctx, fixture.token.Hash)
			if !committed {
				require.NoError(t, err)
				require.Empty(t, pending, "queue and refresh deletion must roll back together")
				return
			}
			require.ErrorIs(t, err, errs.ErrNotFound)
			require.Contains(t, pending, fixture.token.Jti)
			out, err := authRefresh.New(fixture.postgres, fixture.redis, fixture.generator).Refresh(ctx, fixture.input)
			require.ErrorIs(t, err, errs.ErrAuthorizationFailed)
			require.Empty(t, out)
			_, err = cleanupSessions.New(fixture.postgres, fixture.redis).Cleanup(ctx, cleanupSessions.Input{})
			require.NoError(t, err)
			require.ErrorIs(t, fixture.redis.Get(ctx, fixture.token.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound)
		})
	}
}

func TestRepeatedLogoutFinishesPreviouslyQueuedRotatedJtiIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	ctx := logoutContext(t, fixture, fixture.token.Jti)
	out, err := authRefresh.New(fixture.postgres, fixture.redis, fixture.generator).Refresh(ctx, fixture.input)
	require.NoError(t, err)
	claims, err := fixture.generator.GetClaims(out.AccessToken)
	require.NoError(t, err)
	failing := &logoutRedis{
		trackedRefreshRedis: fixture.redis,
		delete: func(context.Context, string) error {
			return errs.ErrServiceUnavailable
		},
	}
	require.ErrorIs(t, authLogout.New(fixture.postgres, failing).Logout(ctx), errs.ErrServiceUnavailable)
	pending, err := fixture.postgres.GetSessionRevocations(ctx, fixture.token.UserID, fixture.token.Audit.Fingerprint)
	require.NoError(t, err)
	require.ElementsMatch(t, []core.Jti{fixture.token.Jti, claims.Jti}, pending)
	require.NoError(t, authLogout.New(fixture.postgres, fixture.redis).Logout(ctx))
	var session core.RedisSession
	require.ErrorIs(t, fixture.redis.Get(ctx, claims.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound,
		"repeat must revoke previous candidate even though refresh row is already absent")
	pending, err = fixture.postgres.GetSessionRevocations(ctx, fixture.token.UserID, fixture.token.Audit.Fingerprint)
	require.NoError(t, err)
	require.Empty(t, pending)
}

func TestRevocationQueueScopeAndKeysetIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	ctx := t.Context()
	otherFingerprint, err := core.NewFingerprint(strings.Repeat("b", 64))
	require.NoError(t, err)
	var jtis []core.Jti
	for _, id := range []string{
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003",
	} {
		jti, err := core.NewJti(uuid.MustParse(id))
		require.NoError(t, err)
		jtis = append(jtis, jti)
	}
	require.NoError(t, fixture.postgres.QueueSessionRevocation(ctx, fixture.token.UserID, fixture.token.Audit.Fingerprint, jtis[0]))
	require.NoError(t, fixture.postgres.QueueSessionRevocation(ctx, fixture.token.UserID, otherFingerprint, jtis[1]))
	require.NoError(t, fixture.postgres.QueueSessionRevocation(ctx, uuid.New(), fixture.token.Audit.Fingerprint, jtis[2]))
	// Repeating a reservation does not add a second task for the same immutable JTI.
	require.NoError(t, fixture.postgres.QueueSessionRevocation(ctx, fixture.token.UserID, fixture.token.Audit.Fingerprint, jtis[0]))
	scoped, err := fixture.postgres.GetSessionRevocations(ctx, fixture.token.UserID, fixture.token.Audit.Fingerprint)
	require.NoError(t, err)
	require.Equal(t, []core.Jti{jtis[0]}, scoped)
	cursor := uuid.Nil
	for _, jti := range jtis {
		page, err := fixture.postgres.GetPendingSessionRevocations(ctx, cursor, limit.MustNew(1))
		require.NoError(t, err)
		require.Equal(t, []core.Jti{jti}, page)
		cursor = uuid.UUID(jti)
		require.NoError(t, fixture.postgres.ConfirmSessionRevocation(ctx, jti))
		require.NoError(t, fixture.postgres.ConfirmSessionRevocation(ctx, jti))
	}
	page, err := fixture.postgres.GetPendingSessionRevocations(ctx, cursor, limit.MustNew(1))
	require.NoError(t, err)
	require.Empty(t, page)
}

type concurrentRevocationPostgres struct {
	*Repository
	ready   chan<- struct{}
	release <-chan struct{}
}

func (p *concurrentRevocationPostgres) GetPendingSessionRevocations(
	ctx context.Context,
	after uuid.UUID,
	batchSize limit.Limit,
) ([]core.Jti, error) {
	pending, err := p.Repository.GetPendingSessionRevocations(ctx, after, batchSize)
	if err != nil {
		return nil, err
	}
	p.ready <- struct{}{}
	select {
	case <-p.release:
		return pending, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestConcurrentSessionCleanupIsIdempotentIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	ctx, cancel := context.WithTimeout(logoutContext(t, fixture, fixture.token.Jti), 10*time.Second)
	defer cancel()
	failing := &logoutRedis{
		trackedRefreshRedis: fixture.redis,
		delete: func(context.Context, string) error {
			return errs.ErrServiceUnavailable
		},
	}
	require.ErrorIs(t, authLogout.New(fixture.postgres, failing).Logout(ctx), errs.ErrServiceUnavailable)
	ready, release := make(chan struct{}, 2), make(chan struct{})
	var once sync.Once
	unlock := func() { once.Do(func() { close(release) }) }
	var workers sync.WaitGroup
	defer func() { cancel(); unlock(); workers.Wait() }()
	postgres := &concurrentRevocationPostgres{Repository: fixture.postgres, ready: ready, release: release}
	done := make(chan error, 2)
	for range 2 {
		workers.Go(func() {
			_, err := cleanupSessions.New(postgres, fixture.redis).Cleanup(ctx, cleanupSessions.Input{})
			done <- err
		})
	}
	for range 2 {
		select {
		case <-ready:
		case <-ctx.Done():
			t.Fatal("cleanup did not read the pending record:", ctx.Err())
		}
	}
	unlock()
	require.NoError(t, <-done)
	require.NoError(t, <-done)
	workers.Wait()
	pending, err := fixture.postgres.GetPendingSessionRevocations(ctx, uuid.Nil, limit.MustNew(100))
	require.NoError(t, err)
	require.Empty(t, pending)
	var session core.RedisSession
	require.ErrorIs(t, fixture.redis.Get(ctx, fixture.token.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound)
}

func TestRevocationSchemaReadinessRejectsMissingTableIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	require.NoError(t, fixture.postgres.checkSessionRevocationsSchema(t.Context()))
	_, err := fixture.postgres.pool.Execx(t.Context(), sq.Expr("DROP TABLE session_revocations"))
	require.NoError(t, err)
	require.Error(t, fixture.postgres.checkSessionRevocationsSchema(t.Context()))
}
