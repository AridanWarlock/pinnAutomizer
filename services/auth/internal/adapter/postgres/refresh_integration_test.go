package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	authRefresh "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/refresh"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestRefreshRetriesRedisFailureBeforeOrAfterServerEffectIntegration(t *testing.T) {
	for _, applied := range []bool{false, true} {
		t.Run(map[bool]string{false: "SET rejected", true: "SET response lost"}[applied], func(t *testing.T) {
			fixture := newRefreshIntegration(t)
			ctx := fixture.token.Audit.WithContext(t.Context())
			failure := errors.New("injected Redis SET failure")
			fixture.redis.setFailure, fixture.redis.applySet = failure, applied
			uc := authRefresh.New(fixture.postgres, fixture.redis, fixture.generator)
			out, err := uc.Refresh(ctx, fixture.input)
			require.ErrorIs(t, err, failure)
			require.ErrorIs(t, err, errs.ErrServiceUnavailable)
			require.Empty(t, out)
			old, err := fixture.postgres.GetRefreshTokenByHash(ctx, fixture.token.Hash)
			require.NoError(t, err, "failed preparation must not consume the old refresh token")
			require.Equal(t, fixture.token.Jti, old.Jti)
			var session core.RedisSession
			require.NoError(t, fixture.redis.Get(ctx, old.Jti.ToRedisKey(), &session))
			fixture.redis.setFailure = nil
			out, err = uc.Refresh(ctx, fixture.input)
			require.NoError(t, err)
			current, err := fixture.postgres.GetRefreshTokenByHash(ctx, crypt.Sha256(out.RefreshTokenString))
			require.NoError(t, err)
			claims, err := fixture.generator.GetClaims(out.AccessToken)
			require.NoError(t, err)
			require.Equal(t, current.Jti, claims.Jti)
			require.Equal(t, fixture.token.ExpiresAt, out.RefreshTokenExpiresAt)
			require.NoError(t, fixture.redis.Get(ctx, claims.Jti.ToRedisKey(), &session))
			require.ErrorIs(t, fixture.redis.Get(ctx, old.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound)
		})
	}
}

type barrierRefreshPostgres struct {
	*Repository
	ready   chan<- struct{}
	release <-chan struct{}
}

func (p barrierRefreshPostgres) GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	token, err := p.Repository.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return domain.RefreshToken{}, err
	}
	p.ready <- struct{}{}
	select {
	case <-p.release:
		return token, nil
	case <-ctx.Done():
		return domain.RefreshToken{}, ctx.Err()
	}
}

func TestConcurrentRefreshHasOneWinnerAndDoesNotDeleteItsSessionIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	ctx, cancel := context.WithTimeout(fixture.token.Audit.WithContext(t.Context()), 10*time.Second)
	ready, release := make(chan struct{}, 2), make(chan struct{})
	var once sync.Once
	unlock := func() { once.Do(func() { close(release) }) }
	postgres := barrierRefreshPostgres{Repository: fixture.postgres, ready: ready, release: release}
	uc := authRefresh.New(postgres, fixture.redis, fixture.generator)
	type result struct {
		out authRefresh.Output
		err error
	}
	done := make(chan result, 2)
	var workers sync.WaitGroup
	defer func() { cancel(); unlock(); workers.Wait() }()
	for range 2 {
		workers.Go(func() {
			out, err := uc.Refresh(ctx, fixture.input)
			done <- result{out: out, err: err}
		})
	}
	for range 2 {
		select {
		case <-ready:
		case got := <-done:
			t.Fatalf("refresh finished before both requests read the old token: %v", got.err)
		case <-ctx.Done():
			t.Fatal("requests did not reach the barrier")
		}
	}
	unlock()
	var winner authRefresh.Output
	successes := 0
	for range 2 {
		got := <-done
		if got.err == nil {
			successes++
			winner = got.out
		} else {
			require.ErrorIs(t, got.err, errs.ErrAuthorizationFailed)
			require.Empty(t, got.out)
		}
	}
	require.Equal(t, 1, successes)
	claims, err := fixture.generator.GetClaims(winner.AccessToken)
	require.NoError(t, err)
	current, err := fixture.postgres.GetRefreshTokenByHash(ctx, crypt.Sha256(winner.RefreshTokenString))
	require.NoError(t, err)
	require.Equal(t, claims.Jti, current.Jti)
	_, err = fixture.postgres.GetRefreshTokenByHash(ctx, fixture.token.Hash)
	require.ErrorIs(t, err, errs.ErrNotFound)
	require.Equal(t, fixture.token.ExpiresAt, current.ExpiresAt)
	var session core.RedisSession
	require.NoError(t, fixture.redis.Get(ctx, claims.Jti.ToRedisKey(), &session))
	require.Equal(t, fixture.token.UserID, session.UserID)
	require.NotContains(t, fixture.redis.deleted, claims.Jti.ToRedisKey(), "loser must not delete winner's session")
	for key := range fixture.redis.keys {
		if key == claims.Jti.ToRedisKey() {
			continue
		}
		require.ErrorIs(t, fixture.redis.Get(ctx, key, &session), errs.ErrKeyNotFound)
	}
}

type lostRotationResponse struct {
	*Repository
	hash string
	jti  core.Jti
}

func (p *lostRotationResponse) RotateRefreshToken(
	ctx context.Context,
	oldHash string,
	newHash string,
	newJti core.Jti,
) (core.Changed, error) {
	changed, err := p.Repository.RotateRefreshToken(ctx, oldHash, newHash, newJti)
	if err != nil || !changed {
		return changed, err
	}
	p.hash, p.jti = newHash, newJti
	return false, errors.New("SQL response lost after commit")
}

func TestUnknownRotationOutcomeDoesNotDeletePossiblyCommittedSessionIntegration(t *testing.T) {
	fixture := newRefreshIntegration(t)
	ctx := fixture.token.Audit.WithContext(t.Context())
	postgres := &lostRotationResponse{Repository: fixture.postgres}
	uc := authRefresh.New(postgres, fixture.redis, fixture.generator)
	out, err := uc.Refresh(ctx, fixture.input)
	require.ErrorContains(t, err, "SQL response lost after commit")
	require.Empty(t, out)
	current, err := fixture.postgres.GetRefreshTokenByHash(ctx, postgres.hash)
	require.NoError(t, err)
	require.Equal(t, postgres.jti, current.Jti)
	var session core.RedisSession
	require.NoError(t, fixture.redis.Get(ctx, postgres.jti.ToRedisKey(), &session))
	_, err = uc.Refresh(ctx, fixture.input)
	require.ErrorIs(t, err, errs.ErrAuthorizationFailed, "old token cannot recover an unknown SQL commit without an idempotency contract")
}
