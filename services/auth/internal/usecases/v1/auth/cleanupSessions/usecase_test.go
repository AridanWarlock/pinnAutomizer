package cleanupSessions

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func cleanupJti(t *testing.T) core.Jti {
	t.Helper()
	jti, err := core.NewJti(uuid.New())
	require.NoError(t, err)
	return jti
}

func TestCleanupContinuesPastFailureAndAdvancesCursor(t *testing.T) {
	t.Parallel()
	first, second := cleanupJti(t), cleanupJti(t)
	failure := errors.New("Redis unavailable")
	postgres := &MockPostgres{
		GetPendingSessionRevocationsFunc: func(
			_ context.Context,
			after uuid.UUID,
			batch limit.Limit,
		) ([]core.Jti, error) {
			require.Equal(t, uuid.Nil, after)
			require.Equal(t, 100, batch.Int())
			return []core.Jti{first, second}, nil
		},
		ConfirmSessionRevocationFunc: func(_ context.Context, jti core.Jti) error {
			require.Equal(t, second, jti, "failed deletion must not be acknowledged")
			return nil
		},
	}
	redis := &MockRedis{
		DeleteFunc: func(_ context.Context, key string) error {
			if key == first.ToRedisKey() {
				return failure
			}
			return errs.ErrKeyNotFound
		},
	}
	uc := New(postgres, redis)
	out, err := uc.Cleanup(t.Context(), Input{})
	require.ErrorIs(t, err, failure)
	require.Equal(t, uuid.UUID(second), out.NextJti)
	require.Len(t, postgres.ConfirmSessionRevocationCalls(), 1)
	postgres.GetPendingSessionRevocationsFunc = func(
		_ context.Context,
		after uuid.UUID,
		_ limit.Limit,
	) ([]core.Jti, error) {
		require.Equal(t, out.NextJti, after)
		return nil, nil
	}
	next, err := uc.Cleanup(t.Context(), Input{AfterJti: out.NextJti})
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, next.NextJti, "empty page restarts sweep, including failed earlier keys")
}

func TestCleanupRetriesAfterAcknowledgmentFailure(t *testing.T) {
	t.Parallel()
	jti := cleanupJti(t)
	failure := errors.New("postgres acknowledgement failed")
	postgres := &MockPostgres{
		GetPendingSessionRevocationsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]core.Jti, error) {
			return []core.Jti{jti}, nil
		},
		ConfirmSessionRevocationFunc: func(context.Context, core.Jti) error {
			return failure
		},
	}
	redis := &MockRedis{
		DeleteFunc: func(context.Context, string) error {
			return nil
		},
	}
	uc := New(postgres, redis)
	_, err := uc.Cleanup(t.Context(), Input{})
	require.ErrorIs(t, err, failure)
	postgres.ConfirmSessionRevocationFunc = func(context.Context, core.Jti) error {
		return nil
	}
	redis.DeleteFunc = func(context.Context, string) error {
		return errs.ErrKeyNotFound
	}
	_, err = uc.Cleanup(t.Context(), Input{})
	require.NoError(t, err, "DEL is safe to repeat after lost acknowledgement")
}

func TestCleanupCancellationDoesNotAcknowledgePendingKeys(t *testing.T) {
	t.Parallel()
	jti := cleanupJti(t)
	ctx, cancel := context.WithCancel(t.Context())
	postgres := &MockPostgres{
		GetPendingSessionRevocationsFunc: func(context.Context, uuid.UUID, limit.Limit) ([]core.Jti, error) {
			cancel()
			return []core.Jti{jti}, nil
		},
	}
	redis := &MockRedis{}
	_, err := New(postgres, redis).Cleanup(ctx, Input{})
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, redis.DeleteCalls())
	require.Empty(t, postgres.ConfirmSessionRevocationCalls())
}
