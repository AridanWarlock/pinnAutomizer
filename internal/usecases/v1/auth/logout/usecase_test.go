package authLogout

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
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
				postgres: &MockPostgres{},
				redis:    &MockRedis{},
			}
			tt.prepare(f)

			ctx := test.ContextWithZeroLogger()
			ctx = fixtures.NewAuditInfo().WithContext(ctx)
			ctx = fixtures.NewAuthInfo().WithContext(ctx)

			uc := New(f.postgres, f.redis)
			err := uc.Logout(ctx)

			tt.check(t, err, f)
		})
	}
}
