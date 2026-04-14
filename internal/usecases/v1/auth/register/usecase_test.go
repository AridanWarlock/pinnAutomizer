package authRegister

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"
)

func TestUsecase_Register(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
		hasher   *MockPasswordHasher
	}

	var (
		fixedUser = fixtures.NewUser()
		testCtx   = test.ContextWithZeroLogger()
	)

	tests := []struct {
		name    string
		input   Input
		prepare func(f *fields)
		check   func(t *testing.T, out Output, err error, f *fields)
	}{
		{
			name: "success path",
			input: Input{
				Login:             "new_user",
				Password:          "password123",
				PasswordConfirmed: "password123",
			},
			prepare: func(f *fields) {
				f.hasher.HashPasswordFunc = func(password string) (string, error) {
					return "hashed_password", nil
				}
				f.postgres.GetRoleByTitleFunc = func(ctx context.Context, title string) (domain.Role, error) {
					return fixtures.NewRole(), nil
				}

				f.postgres.InTransactionFunc = func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}
				f.postgres.CreateUserFunc = func(ctx context.Context, user domain.User) (domain.User, error) {
					return fixedUser, nil
				}
				f.postgres.CreateUsersRolesBatchFunc = func(ctx context.Context, usersRoles []domain.UsersRoles) ([]domain.UsersRoles, error) {
					return usersRoles, nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, fixedUser, out.User)

				assert.Len(t, f.hasher.HashPasswordCalls(), 1)

				sentUser := f.postgres.CreateUserCalls()[0].User
				assert.Equal(t, "hashed_password", sentUser.PasswordHash)
			},
		},
		{
			name: "error - hashing failed",
			input: Input{
				Login:             "user",
				Password:          "password",
				PasswordConfirmed: "password",
			},
			prepare: func(f *fields) {
				f.hasher.HashPasswordFunc = func(password string) (string, error) {
					return "", errs.ErrInvalidCredentials
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrInvalidCredentials))
				assert.Len(t, f.postgres.CreateUserCalls(), 0)
			},
		},
		{
			name: "error - password mismatch",
			input: Input{
				Login:             "user",
				Password:          "password",
				PasswordConfirmed: "wrong_password",
			},
			prepare: func(f *fields) {},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrInvalidArgument))
				assert.Len(t, f.postgres.CreateUserCalls(), 0)
			},
		},
		{
			name: "error - login already taken",
			input: Input{
				Login:             "existing_admin",
				Password:          "password",
				PasswordConfirmed: "password",
			},
			prepare: func(f *fields) {
				f.hasher.HashPasswordFunc = func(password string) (string, error) {
					return "hash", nil
				}
				f.postgres.GetRoleByTitleFunc = func(ctx context.Context, title string) (domain.Role, error) {
					return fixtures.NewRole(), nil
				}

				f.postgres.InTransactionFunc = func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}
				f.postgres.CreateUserFunc = func(ctx context.Context, user domain.User) (domain.User, error) {
					return domain.User{}, errs.ErrConflict
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrConflict))
			},
		},
		{
			name: "error - database failure on createUsersRolesBatch",
			input: Input{
				Login:             "admin",
				Password:          "password",
				PasswordConfirmed: "password",
			},
			prepare: func(f *fields) {
				f.hasher.HashPasswordFunc = func(password string) (string, error) {
					return "hash", nil
				}
				f.postgres.GetRoleByTitleFunc = func(ctx context.Context, title string) (domain.Role, error) {
					return fixtures.NewRole(), nil
				}

				f.postgres.InTransactionFunc = func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}
				f.postgres.CreateUserFunc = func(ctx context.Context, user domain.User) (domain.User, error) {
					return fixedUser, nil
				}
				f.postgres.CreateUsersRolesBatchFunc = func(ctx context.Context, usersRoles []domain.UsersRoles) ([]domain.UsersRoles, error) {
					return nil, sql.ErrConnDone
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
				hasher:   &MockPasswordHasher{},
			}
			tt.prepare(f)

			uc := New(f.postgres, f.hasher)
			out, err := uc.Register(testCtx, tt.input)

			tt.check(t, out, err, f)
		})
	}
}
