package authLogin

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestUsecase_Login(t *testing.T) {
	type fields struct {
		postgres         *MockPostgres
		accessGenerator  *MockAccessTokenGenerator
		refreshGenerator *MockRefreshTokenGenerator
		hasher           *MockPasswordHasher
		clock            *MockClock
	}

	var (
		fixedID          = uuid.MustParse("00000000-0000-0000-0000-000000000001")
		fixedNow         = time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
		fixedToken       = fixtures.NewAccessToken()
		fixedFingerprint = fixtures.NewFingerprint()
		testCtx          = test.ContextBackgroundWithZeroLogger()
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
				Login:       "admin",
				Password:    "12345678",
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				user := fixtures.NewUser(func(user *domain.User) {
					user.ID = fixedID
					user.Login = "admin"
				})
				roles := []domain.Role{fixtures.NewRole()}
				refreshToken := fixtures.NewRefreshToken(func(refreshToken *domain.RefreshToken) {
					refreshToken.CreatedAt = fixedNow
					refreshToken.ExpiresAt = fixedNow.Add(time.Hour)
				})
				session := fixtures.NewUserSession(func(session *domain.UserSession) {
					session.Fingerprint = fixedFingerprint
				})

				f.postgres.GetUserByLoginFunc = func(ctx context.Context, login string) (domain.User, error) {
					return user, nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error {
					return nil
				}
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, uID uuid.UUID) ([]domain.Role, error) {
					return roles, nil
				}
				f.accessGenerator.GenerateFunc = func(u domain.User, r []domain.Role, f domain.Fingerprint) (domain.AccessToken, error) {
					return fixedToken, nil
				}
				f.refreshGenerator.GenerateFunc = func() (domain.RefreshToken, error) {
					return refreshToken, nil
				}
				f.clock.NowFunc = func() time.Time {
					return fixedNow
				}
				f.postgres.LoginFunc = func(ctx context.Context, s domain.UserSession) (domain.UserSession, error) {
					return session, nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, fixedToken, out.AccessToken)
				assert.Equal(t, fixedNow.Add(time.Hour), out.RefreshTokenExpiresAt)
			},
		},
		{
			name: "error - invalid argument (empty login)",
			input: Input{
				Login:       "",
				Password:    "12345678",
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrInvalidArgument))
				assert.Len(t, f.postgres.GetUserByLoginCalls(), 0)
			},
		},
		{
			name: "error - invalid credentials (user not found)",
			input: Input{
				Login:       "unknown",
				Password:    "12345678",
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, l string) (domain.User, error) {
					return domain.User{}, errs.ErrNotFound
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid credentials")

				assert.Len(t, f.hasher.CompareHashAndPasswordCalls(), 0)
			},
		},
		{
			name: "error - invalid credentials (invalid password)",
			input: Input{
				Login:       "admin",
				Password:    "12345678",
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				user := fixtures.NewUser(func(user *domain.User) {
					user.Login = "user"
				})

				f.postgres.GetUserByLoginFunc = func(ctx context.Context, l string) (domain.User, error) {
					return user, nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(hash string, password string) error {
					return errors.New("invalid password")
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid credentials")

				assert.Len(t, f.accessGenerator.GenerateCalls(), 0)
			},
		},
		{
			name: "error - access token generation failed",
			input: Input{
				Login:       "admin",
				Password:    "12345678",
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, l string) (domain.User, error) {
					return fixtures.NewUser(), nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error { return nil }
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, id uuid.UUID) ([]domain.Role, error) {
					return []domain.Role{fixtures.NewRole()}, nil
				}
				f.accessGenerator.GenerateFunc = func(u domain.User, r []domain.Role, f domain.Fingerprint) (domain.AccessToken, error) {
					return "", errors.New("crypto: not available")
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "crypto: not available")
				assert.Len(t, f.refreshGenerator.GenerateCalls(), 0)
			},
		},
		{
			name: "error - refresh token generation failed",
			input: Input{
				Login:       "admin",
				Password:    "12345678",
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, l string) (domain.User, error) {
					return fixtures.NewUser(), nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error { return nil }
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, id uuid.UUID) ([]domain.Role, error) {
					return []domain.Role{fixtures.NewRole()}, nil
				}
				f.accessGenerator.GenerateFunc = func(u domain.User, r []domain.Role, f domain.Fingerprint) (domain.AccessToken, error) {
					return fixedToken, nil
				}
				f.refreshGenerator.GenerateFunc = func() (domain.RefreshToken, error) {
					return domain.RefreshToken{}, errors.New("refresh generator failed")
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "refresh generator failed")
				assert.Len(t, f.postgres.LoginCalls(), 0)
			},
		},
		{
			name: "error - database failure on GetRoles",
			input: Input{
				Login:       "admin",
				Password:    "12345678",
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, l string) (domain.User, error) {
					return fixtures.NewUser(), nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error { return nil }
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, id uuid.UUID) ([]domain.Role, error) {
					return nil, sql.ErrConnDone
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), sql.ErrConnDone.Error())
				assert.Len(t, f.accessGenerator.GenerateCalls(), 0)
			},
		},
		{
			name: "error - database failure on saving session (Login)",
			input: Input{
				Login:       "admin",
				Password:    "12345678",
				Fingerprint: fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, l string) (domain.User, error) {
					return fixtures.NewUser(), nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error { return nil }
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, id uuid.UUID) ([]domain.Role, error) {
					return []domain.Role{fixtures.NewRole()}, nil
				}
				f.accessGenerator.GenerateFunc = func(u domain.User, r []domain.Role, f domain.Fingerprint) (domain.AccessToken, error) {
					return fixedToken, nil
				}
				f.refreshGenerator.GenerateFunc = func() (domain.RefreshToken, error) {
					return fixtures.NewRefreshToken(), nil
				}
				f.clock.NowFunc = func() time.Time { return fixedNow }
				f.postgres.LoginFunc = func(ctx context.Context, s domain.UserSession) (domain.UserSession, error) {
					return domain.UserSession{}, errors.New("unique constraint violation or deadlock")
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unique constraint violation or deadlock")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				postgres:         &MockPostgres{},
				accessGenerator:  &MockAccessTokenGenerator{},
				refreshGenerator: &MockRefreshTokenGenerator{},
				hasher:           &MockPasswordHasher{},
				clock:            &MockClock{},
			}

			tt.prepare(f)

			uc := New(f.postgres, f.accessGenerator, f.refreshGenerator, f.hasher, f.clock)
			out, err := uc.Login(testCtx, tt.input)

			tt.check(t, out, err, f)
		})
	}
}
