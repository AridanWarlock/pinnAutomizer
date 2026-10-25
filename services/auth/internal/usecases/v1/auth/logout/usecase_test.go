package authLogout

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
)

func TestUsecase_Logout(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
		redis    *MockRedis
	}

	tests := []struct {
		name    string
		prepare func(f *fields)
		check   func(t *testing.T, err error, f *fields)
	}{
		{
			name: "successful path with existing access and refresh",
			prepare: func(f *fields) {
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return nil
				}
				f.postgres.LogoutFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) error {
					return nil
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.NoError(t, err)
			},
		},
		{
			name: "successful path with existing access and no existing refresh",
			prepare: func(f *fields) {
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return nil
				}
				f.postgres.LogoutFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) error {
					return errs.ErrNotFound
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.NoError(t, err)
			},
		},
		{
			name: "successful path with no existing access and existing refresh",
			prepare: func(f *fields) {
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return errs.ErrKeyNotFound
				}
				f.postgres.LogoutFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) error {
					return nil
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.NoError(t, err)
			},
		},
		{
			name: "successful path without existing access and refresh",
			prepare: func(f *fields) {
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return errs.ErrKeyNotFound
				}
				f.postgres.LogoutFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) error {
					return errs.ErrNotFound
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.NoError(t, err)
			},
		},
		{
			name: "error - redis failure on deleting session",
			prepare: func(f *fields) {
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return errs.ErrClosed
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrClosed))
			},
		},
		{
			name: "error - db failure on deleting session",
			prepare: func(f *fields) {
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return nil
				}
				f.postgres.LogoutFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) error {
					return sql.ErrConnDone
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, sql.ErrConnDone))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				postgres: &MockPostgres{
					QueueSessionRevocationFunc: func(context.Context, uuid.UUID, core.Fingerprint, core.Jti) error {
						return nil
					},
					GetSessionRevocationsFunc: func(ctx context.Context, _ uuid.UUID, _ core.Fingerprint) ([]core.Jti, error) {
						return []core.Jti{core.MustAuthInfoFromContext(ctx).Jti}, nil
					},
					ConfirmSessionRevocationFunc: func(context.Context, core.Jti) error {
						return nil
					},
					LogoutFunc: func(context.Context, uuid.UUID, core.Fingerprint) error {
						return nil
					},

					InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					},
					GetJtiByFingerprintForUpdateFunc: func(
						ctx context.Context,
						_ uuid.UUID,
						_ core.Fingerprint,
					) (core.Jti, error) {
						return core.MustAuthInfoFromContext(ctx).Jti, nil
					},
				},
				redis: &MockRedis{},
			}
			tt.prepare(f)

			ctx := test.ContextWithZeroLogger()
			ctx = corefixtures.NewAuditInfo().WithContext(ctx)
			ctx = corefixtures.NewAuthInfo().WithContext(ctx)

			uc := New(f.postgres, f.redis)
			err := uc.Logout(ctx)

			tt.check(t, err, f)
		})
	}
}

func TestLogoutCommitsRevocationBeforeDeletingRedisSessions(t *testing.T) {
	t.Parallel()
	auth := corefixtures.NewAuthInfo()
	audit := corefixtures.NewAuditInfo()
	current, err := core.NewJti(uuid.New())
	require.NoError(t, err)
	ctx := auth.WithContext(audit.WithContext(t.Context()))
	operations := []string{}
	postgres := &MockPostgres{
		QueueSessionRevocationFunc: func(_ context.Context, _ uuid.UUID, _ core.Fingerprint, jti core.Jti) error {
			operations = append(operations, "queue "+jti.ToRedisKey())
			return nil
		},
		GetSessionRevocationsFunc: func(context.Context, uuid.UUID, core.Fingerprint) ([]core.Jti, error) {
			return []core.Jti{current, auth.Jti}, nil
		},
		ConfirmSessionRevocationFunc: func(_ context.Context, jti core.Jti) error {
			operations = append(operations, "ack "+jti.ToRedisKey())
			return nil
		},
		InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			operations = append(operations, "begin")
			if err := fn(ctx); err != nil {
				return err
			}
			operations = append(operations, "commit")
			return nil
		},
		GetJtiByFingerprintForUpdateFunc: func(
			_ context.Context,
			userID uuid.UUID,
			fingerprint core.Fingerprint,
		) (core.Jti, error) {
			require.Equal(t, auth.UserID, userID)
			require.Equal(t, audit.Fingerprint, fingerprint)
			operations = append(operations, "lock")
			return current, nil
		},
		LogoutFunc: func(context.Context, uuid.UUID, core.Fingerprint) error {
			operations = append(operations, "delete refresh")
			return nil
		},
	}
	redis := &MockRedis{
		DeleteFunc: func(_ context.Context, key string) error {
			operations = append(operations, key)
			return nil
		},
	}
	require.NoError(t, New(postgres, redis).Logout(ctx))
	require.Equal(t, []string{
		"begin", "lock", "queue " + current.ToRedisKey(), "queue " + auth.Jti.ToRedisKey(), "delete refresh", "commit",
		current.ToRedisKey(), "ack " + current.ToRedisKey(), auth.Jti.ToRedisKey(), "ack " + auth.Jti.ToRedisKey(),
	}, operations)
}

func TestLogoutLockFailureDoesNotDeleteRedisSession(t *testing.T) {
	t.Parallel()
	failure := errors.New("postgres unavailable")
	postgres := &MockPostgres{
		InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
		GetJtiByFingerprintForUpdateFunc: func(
			context.Context,
			uuid.UUID,
			core.Fingerprint,
		) (core.Jti, error) {
			return core.Jti{}, failure
		},
	}
	redis := &MockRedis{}
	ctx := corefixtures.NewAuthInfo().WithContext(corefixtures.NewAuditInfo().WithContext(t.Context()))
	require.ErrorIs(t, New(postgres, redis).Logout(ctx), failure)
	require.Empty(t, redis.DeleteCalls())
	require.Empty(t, postgres.LogoutCalls())
}
