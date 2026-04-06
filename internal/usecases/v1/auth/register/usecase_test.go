package auth_register

import (
	"context"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pg_errors"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestUsecase_Register(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
		hasher   *MockPasswordHasher
	}

	tests := []struct {
		name    string
		input   Input
		prepare func(f fields)
		wantErr bool
	}{
		{
			name: "valid path",
			input: Input{
				Login:             "newuser",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
				f.hasher.EXPECT().
					HashPassword("12345678").
					Return("hashed", nil).Once()

				f.postgres.EXPECT().
					GetRoleByTitle(mock.Anything, "ROLE_USER").
					Return(domain.Role{
						ID:    uuid.Max,
						Title: "ROLE_USER",
					}, nil).Once()

				var createdUser domain.User

				f.postgres.EXPECT().
					CreateUser(mock.Anything, mock.MatchedBy(func(user domain.User) bool {
						return user.PasswordHash == "hashed" && user.Login == "newuser"
					})).
					Run(func(ctx context.Context, user domain.User) {
						createdUser = user
					}).Return(createdUser, nil).Once()

				f.postgres.EXPECT().
					CreateAuthToken(mock.Anything, createdUser.ID).
					Return(domain.AuthToken{
						UserID:       createdUser.ID,
						AccessToken:  "access",
						RefreshToken: "refresh",
					}, nil).Once()

				usersRoles := []domain.UsersRoles{{
					UserID: createdUser.ID,
					RoleID: uuid.Max,
				}}
				f.postgres.EXPECT().
					CreateUsersRolesBatch(mock.Anything, usersRoles).
					Return(usersRoles, nil).Once()

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					}).Once()
			},
			wantErr: false,
		},
		{
			name: "rollback tx",
			input: Input{
				Login:             "newuser",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
				f.hasher.EXPECT().
					HashPassword("12345678").
					Return("hashed", nil).Once()

				f.postgres.EXPECT().
					GetRoleByTitle(mock.Anything, "ROLE_USER").
					Return(domain.Role{
						ID:    uuid.Max,
						Title: "ROLE_USER",
					}, nil).Once()

				var createdUser domain.User

				f.postgres.EXPECT().
					CreateUser(mock.Anything, mock.MatchedBy(func(user domain.User) bool {
						return user.PasswordHash == "hashed" && user.Login == "newuser"
					})).
					Run(func(ctx context.Context, user domain.User) {
						createdUser = user
					}).Return(createdUser, nil).Once()

				f.postgres.EXPECT().
					CreateAuthToken(mock.Anything, createdUser.ID).
					Return(domain.AuthToken{
						UserID:       createdUser.ID,
						AccessToken:  "access",
						RefreshToken: "refresh",
					}, nil).Once()

				usersRoles := []domain.UsersRoles{{
					UserID: createdUser.ID,
					RoleID: uuid.Max,
				}}
				f.postgres.EXPECT().
					CreateUsersRolesBatch(mock.Anything, usersRoles).
					Return(usersRoles, nil).Once()

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						_ = fn(ctx)
						return pgx.ErrTxCommitRollback
					}).Once()
			},
			wantErr: true,
		},
		{
			name: "batch insert users_roles failed",
			input: Input{
				Login:             "newuser",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
				f.hasher.EXPECT().
					HashPassword("12345678").
					Return("hashed", nil).Once()

				f.postgres.EXPECT().
					GetRoleByTitle(mock.Anything, "ROLE_USER").
					Return(domain.Role{
						ID:    uuid.Max,
						Title: "ROLE_USER",
					}, nil).Once()

				var createdUser domain.User

				f.postgres.EXPECT().
					CreateUser(mock.Anything, mock.MatchedBy(func(user domain.User) bool {
						return user.PasswordHash == "hashed" && user.Login == "newuser"
					})).
					Run(func(ctx context.Context, user domain.User) {
						createdUser = user
					}).Return(createdUser, nil).Once()

				f.postgres.EXPECT().
					CreateAuthToken(mock.Anything, createdUser.ID).
					Return(domain.AuthToken{
						UserID:       createdUser.ID,
						AccessToken:  "access",
						RefreshToken: "refresh",
					}, nil).Once()

				usersRoles := []domain.UsersRoles{{
					UserID: createdUser.ID,
					RoleID: uuid.Max,
				}}
				f.postgres.EXPECT().
					CreateUsersRolesBatch(mock.Anything, usersRoles).
					Return(nil, pg_errors.ErrInvalidBatchSize).Once()

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					}).Once()
			},
			wantErr: true,
		},
		{
			name: "insert auth_tokens failed",
			input: Input{
				Login:             "newuser",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
				f.hasher.EXPECT().
					HashPassword("12345678").
					Return("hashed", nil).Once()

				f.postgres.EXPECT().
					GetRoleByTitle(mock.Anything, "ROLE_USER").
					Return(domain.Role{
						ID:    uuid.Max,
						Title: "ROLE_USER",
					}, nil).Once()

				var createdUser domain.User

				f.postgres.EXPECT().
					CreateUser(mock.Anything, mock.MatchedBy(func(user domain.User) bool {
						return user.PasswordHash == "hashed" && user.Login == "newuser"
					})).
					Run(func(ctx context.Context, user domain.User) {
						createdUser = user
					}).Return(createdUser, nil).Once()

				f.postgres.EXPECT().
					CreateAuthToken(mock.Anything, createdUser.ID).
					Return(domain.AuthToken{}, pgx.ErrTxClosed).Once()

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					}).Once()
			},
			wantErr: true,
		},
		{
			name: "insert user failed",
			input: Input{
				Login:             "newuser",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
				f.hasher.EXPECT().
					HashPassword("12345678").
					Return("hashed", nil).Once()

				f.postgres.EXPECT().
					GetRoleByTitle(mock.Anything, "ROLE_USER").
					Return(domain.Role{
						ID:    uuid.Max,
						Title: "ROLE_USER",
					}, nil).Once()

				f.postgres.EXPECT().
					CreateUser(mock.Anything, mock.Anything).
					Return(domain.User{}, pgx.ErrTxClosed).Once()

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					}).Once()
			},
			wantErr: true,
		},
		{
			name: "get role failed",
			input: Input{
				Login:             "newuser",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
				f.hasher.EXPECT().
					HashPassword("12345678").
					Return("hashed", nil).Once()

				f.postgres.EXPECT().
					GetRoleByTitle(mock.Anything, "ROLE_USER").
					Return(domain.Role{}, pgx.ErrTxClosed).Once()
			},
			wantErr: true,
		},
		{
			name: "hash password failed",
			input: Input{
				Login:             "newuser",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
				f.hasher.EXPECT().
					HashPassword("12345678").
					Return("", bcrypt.ErrPasswordTooLong).Once()
			},
			wantErr: true,
		},
		{
			name: "small password",
			input: Input{
				Login:             "newuser",
				Password:          "42",
				PasswordConfirmed: "42",
			},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
		{
			name: "not equals password",
			input: Input{
				Login:             "newuser",
				Password:          "42",
				PasswordConfirmed: "24",
			},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
		{
			name: "small login",
			input: Input{
				Login:             "us",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
		{
			name: "invalid login",
			input: Input{
				Login:             " __invalid__ ",
				Password:          "12345678",
				PasswordConfirmed: "12345678",
			},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				postgres: NewMockPostgres(t),
				hasher:   NewMockPasswordHasher(t),
			}
			tt.prepare(f)

			uc := New(f.postgres, f.hasher, zerolog.Logger{})
			err := uc.Register(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)
		})
	}
}
