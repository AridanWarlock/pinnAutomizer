package authLogout

import (
	"context"
	"database/sql"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestUsecase_Logout(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
	}

	var (
		fixedID          = uuid.New()
		fixedFingerprint = fixtures.NewFingerprint()
		testCtx          = test.ContextBackgroundWithZeroLogger()
	)

	tests := []struct {
		name    string
		input   Input
		prepare func(f *fields)
		check   func(t *testing.T, err error, f *fields)
	}{
		{
			name: "successful path",
			input: Input{
				UserID:      fixedID,
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.LogoutFunc = func(ctx context.Context, userID uuid.UUID, fingerprint domain.Fingerprint) error {
					return nil
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.NoError(t, err)
			},
		},
		{
			name: "error - invalid id",
			input: Input{
				UserID:      uuid.Nil,
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errs.ErrInvalidArgument.Error())
				assert.Len(t, f.postgres.LogoutCalls(), 0)
			},
		},
		{
			name: "error - invalid fingerprint",
			input: Input{
				UserID:      fixedID,
				Fingerprint: fixedFingerprint[:31],
			},
			prepare: func(f *fields) {
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errs.ErrInvalidArgument.Error())
				assert.Len(t, f.postgres.LogoutCalls(), 0)
			},
		},
		{
			name: "session already deleted",
			input: Input{
				UserID:      fixedID,
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.LogoutFunc = func(ctx context.Context, userID uuid.UUID, fingerprint domain.Fingerprint) error {
					return errs.ErrNotFound
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.NoError(t, err)
			},
		},
		{
			name: "error - database failure on deleting session",
			input: Input{
				UserID:      fixedID,
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.LogoutFunc = func(ctx context.Context, userID uuid.UUID, fingerprint domain.Fingerprint) error {
					return sql.ErrConnDone
				}
			},
			check: func(t *testing.T, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), sql.ErrConnDone.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				postgres: &MockPostgres{},
			}
			tt.prepare(f)

			uc := New(f.postgres)
			err := uc.Logout(testCtx, tt.input)

			tt.check(t, err, f)
		})
	}
}
