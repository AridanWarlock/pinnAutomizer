package authMe

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestUsecase_Me(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
	}

	tests := []struct {
		name    string
		prepare func(f *fields)
		check   func(t *testing.T, out Output, err error, f *fields)
	}{
		{
			name: "successful path",
			prepare: func(f *fields) {
				f.postgres.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.User, error) {
					return fixtures.NewUser(func(user *domain.User) {
						user.Login = "Ivan Ivanov"
					}), nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, "Ivan Ivanov", out.Login)
			},
		},
		{
			name: "error - user not found",
			prepare: func(f *fields) {
				f.postgres.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.User, error) {
					return domain.User{}, errs.ErrNotFound
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrNotFound))
			},
		},
		{
			name: "error - db failure on get user",
			prepare: func(f *fields) {
				f.postgres.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.User, error) {
					return domain.User{}, sql.ErrConnDone
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, sql.ErrConnDone))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				postgres: &MockPostgres{},
			}

			tt.prepare(f)

			ctx := test.ContextWithZeroLogger()
			ctx = fixtures.NewAuthInfo().WithContext(ctx)

			uc := New(f.postgres)
			actual, err := uc.Me(ctx)

			tt.check(t, actual, err, f)
		})
	}
}
