package authLogin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain/domainfixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestUsecase_Login(t *testing.T) {
	type fields struct {
		postgres       *MockPostgres
		redis          *MockRedis
		tokenGenerator *MockTokenGenerator
		hasher         *MockPasswordHasher
	}

	var (
		fixedID       = uuid.New()
		fixedLogin    = "Ivan Ivanov"
		fixedPassword = "valid_password&1"
		fixedUser     = domainfixtures.NewUser(func(user *domain.User) {
			user.ID = fixedID
			user.Login = fixedLogin
		})
		fixedRoles       = []core.Role{corefixtures.NewRole()}
		fixedNow         = time.Now()
		fixedToken       = corefixtures.NewAccessToken()
		fixedFingerprint = corefixtures.NewFingerprint()
		testCtx          = test.ContextWithZeroLogger()
		fixedJwtClaims   = corefixtures.NewJwtClaims(func(claims *core.JwtClaims) {
			claims.UserID = fixedID
			claims.IssuedAt = fixedNow.Add(-time.Minute)
			claims.Jti = corefixtures.NewJti()
		})
		fixedAudit = corefixtures.NewAuditInfo(func(audit *core.AuditInfo) {
			audit.Fingerprint = fixedFingerprint
		})
	)

	tests := []struct {
		name    string
		input   Input
		prepare func(f *fields)
		check   func(t *testing.T, out Output, err error, f *fields)
	}{
		{
			name: "success path with existing old refresh",
			input: Input{
				Login:    fixedLogin,
				Password: fixedPassword,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, login string) (domain.User, error) {
					return fixedUser, nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error {
					return nil
				}
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, uID uuid.UUID) ([]core.Role, error) {
					return fixedRoles, nil
				}
				f.postgres.GetJtiByFingerprintFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) (core.Jti, error) {
					return corefixtures.NewJti(), nil
				}
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return nil
				}
				f.tokenGenerator.GenerateAndGetClaimsFunc = func(userID uuid.UUID) (core.AccessToken, core.JwtClaims, error) {
					return fixedToken, fixedJwtClaims, nil
				}
				f.postgres.LoginFunc = func(ctx context.Context, refresh domain.RefreshToken) (domain.RefreshToken, error) {
					return refresh, nil
				}
				f.redis.SetFunc = func(ctx context.Context, key string, value any, ttl time.Duration) error {
					return nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, fixedToken, out.AccessToken)
				assert.Less(t, fixedNow, out.RefreshTokenExpiresAt)
			},
		},
		{
			name: "success path without old refresh",
			input: Input{
				Login:    fixedLogin,
				Password: fixedPassword,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, login string) (domain.User, error) {
					return fixedUser, nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error {
					return nil
				}
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, uID uuid.UUID) ([]core.Role, error) {
					return fixedRoles, nil
				}
				f.postgres.GetJtiByFingerprintFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) (core.Jti, error) {
					return core.Jti{}, errs.ErrNotFound
				}
				f.tokenGenerator.GenerateAndGetClaimsFunc = func(userID uuid.UUID) (core.AccessToken, core.JwtClaims, error) {
					return fixedToken, fixedJwtClaims, nil
				}
				f.postgres.LoginFunc = func(ctx context.Context, refresh domain.RefreshToken) (domain.RefreshToken, error) {
					return refresh, nil
				}
				f.redis.SetFunc = func(ctx context.Context, key string, value any, ttl time.Duration) error {
					return nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, fixedToken, out.AccessToken)
				assert.Less(t, fixedNow, out.RefreshTokenExpiresAt)
			},
		},
		{
			name: "success path with old refresh and without old access",
			input: Input{
				Login:    fixedLogin,
				Password: fixedPassword,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, login string) (domain.User, error) {
					return fixedUser, nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error {
					return nil
				}
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, uID uuid.UUID) ([]core.Role, error) {
					return fixedRoles, nil
				}
				f.postgres.GetJtiByFingerprintFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) (core.Jti, error) {
					return corefixtures.NewJti(), nil
				}
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return errs.ErrKeyNotFound
				}
				f.tokenGenerator.GenerateAndGetClaimsFunc = func(userID uuid.UUID) (core.AccessToken, core.JwtClaims, error) {
					return fixedToken, fixedJwtClaims, nil
				}
				f.postgres.LoginFunc = func(ctx context.Context, refresh domain.RefreshToken) (domain.RefreshToken, error) {
					return refresh, nil
				}
				f.redis.SetFunc = func(ctx context.Context, key string, value any, ttl time.Duration) error {
					return nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, fixedToken, out.AccessToken)
				assert.Less(t, fixedNow, out.RefreshTokenExpiresAt)
			},
		},
		{
			name: "error - invalid argument (empty login)",
			input: Input{
				Login:    "",
				Password: fixedPassword,
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
			name: "error - invalid argument (small password)",
			input: Input{
				Login:    fixedLogin,
				Password: "root",
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
				Login:    "unknown",
				Password: fixedPassword,
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
				Login:    fixedLogin,
				Password: "invalid_password",
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, l string) (domain.User, error) {
					return fixedUser, nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(hash string, password string) error {
					return errors.New("invalid password")
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid credentials")

				assert.Len(t, f.postgres.GetRolesByUserIDCalls(), 0)
			},
		},
		{
			name: "error - access token generation failed",
			input: Input{
				Login:    fixedLogin,
				Password: fixedPassword,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByLoginFunc = func(ctx context.Context, l string) (domain.User, error) {
					return fixedUser, nil
				}
				f.hasher.CompareHashAndPasswordFunc = func(h, p string) error { return nil }
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, id uuid.UUID) ([]core.Role, error) {
					return fixedRoles, nil
				}
				f.postgres.GetJtiByFingerprintFunc = func(ctx context.Context, userID uuid.UUID, fingerprint core.Fingerprint) (core.Jti, error) {
					return corefixtures.NewJti(), nil
				}
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return nil
				}
				f.tokenGenerator.GenerateAndGetClaimsFunc = func(userID uuid.UUID) (core.AccessToken, core.JwtClaims, error) {
					return "", core.JwtClaims{}, errors.New("crypto: not available")
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "crypto: not available")
				assert.Len(t, f.postgres.LoginCalls(), 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				postgres:       &MockPostgres{},
				redis:          &MockRedis{},
				tokenGenerator: &MockTokenGenerator{},
				hasher:         &MockPasswordHasher{},
			}

			ctx := fixedAudit.WithContext(testCtx)

			tt.prepare(f)

			uc := New(f.postgres, f.redis, f.tokenGenerator, f.hasher)
			out, err := uc.Login(ctx, tt.input)

			tt.check(t, out, err, f)
		})
	}
}
