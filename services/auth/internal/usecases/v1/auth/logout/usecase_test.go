package authLogout

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestLogoutPersistsRevocationBeforeRedisCleanup(t *testing.T) {
	t.Parallel()
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing Redis session", true: "missing Redis session"}[missing], func(t *testing.T) {
			auth := corefixtures.NewAuthInfo()
			audit := corefixtures.NewAuditInfo()
			ctx := auth.WithContext(audit.WithContext(t.Context()))
			committed := false
			postgres := &MockPostgres{
				RevokeSessionsFunc: func(_ context.Context, userID uuid.UUID, fingerprint core.Fingerprint, jti core.Jti) error {
					require.Equal(t, auth.UserID, userID)
					require.Equal(t, audit.Fingerprint, fingerprint)
					require.Equal(t, auth.Jti, jti)
					committed = true
					return nil
				},
				GetSessionRevocationJtisFunc: func(context.Context, uuid.UUID, core.Fingerprint) ([]core.Jti, error) {
					require.True(t, committed)
					return []core.Jti{auth.Jti}, nil
				},
				ConfirmSessionRevocationFunc: func(_ context.Context, jti core.Jti) error {
					require.Equal(t, auth.Jti, jti)
					return nil
				},
			}
			redis := &MockRedis{
				DeleteFunc: func(_ context.Context, key string) error {
					require.True(t, committed)
					require.Equal(t, auth.Jti.ToRedisKey(), key)
					if missing {
						return errs.ErrKeyNotFound
					}
					return nil
				},
			}
			require.NoError(t, New(postgres, redis).Logout(ctx))
			require.Len(t, postgres.ConfirmSessionRevocationCalls(), 1)
		})
	}
}

func TestLogoutCommitFailureDoesNotTouchRedis(t *testing.T) {
	t.Parallel()
	failure := errors.New("SQL commit outcome unknown")
	postgres := &MockPostgres{
		RevokeSessionsFunc: func(context.Context, uuid.UUID, core.Fingerprint, core.Jti) error { return failure },
	}
	redis := &MockRedis{}
	ctx := corefixtures.NewAuthInfo().WithContext(corefixtures.NewAuditInfo().WithContext(t.Context()))
	require.ErrorIs(t, New(postgres, redis).Logout(ctx), failure)
	require.Empty(t, redis.DeleteCalls())
	require.Empty(t, postgres.GetSessionRevocationJtisCalls())
}

func TestLogoutCleanupFailureLeavesPendingWork(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"list", "Redis DEL", "SQL acknowledgement"} {
		t.Run(phase, func(t *testing.T) {
			auth := corefixtures.NewAuthInfo()
			ctx := auth.WithContext(corefixtures.NewAuditInfo().WithContext(t.Context()))
			failure := errors.New(phase)
			postgres := &MockPostgres{
				RevokeSessionsFunc: func(context.Context, uuid.UUID, core.Fingerprint, core.Jti) error { return nil },
				GetSessionRevocationJtisFunc: func(context.Context, uuid.UUID, core.Fingerprint) ([]core.Jti, error) {
					if phase == "list" {
						return nil, failure
					}
					return []core.Jti{auth.Jti}, nil
				},
				ConfirmSessionRevocationFunc: func(context.Context, core.Jti) error { return failure },
			}
			redis := &MockRedis{
				DeleteFunc: func(context.Context, string) error {
					if phase == "Redis DEL" {
						return failure
					}
					return nil
				},
			}
			require.ErrorIs(t, New(postgres, redis).Logout(ctx), failure)
			if phase != "SQL acknowledgement" {
				require.Empty(t, postgres.ConfirmSessionRevocationCalls())
			}
		})
	}
}
