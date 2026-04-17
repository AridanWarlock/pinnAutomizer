package authRefresh

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain/domainfixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/corefixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUsecase_Refresh(t *testing.T) {
	type fields struct {
		postgres       *MockPostgres
		redis          *MockRedis
		tokenGenerator *MockTokenGenerator
	}

	var (
		validTokenStr = crypt.GenerateSecureToken()

		fixedUserID      = uuid.New()
		fixedRoles       = []core.Role{corefixtures.NewRole()}
		fixedAccessToken = corefixtures.NewAccessToken()
		fixedAuditInfo   = corefixtures.NewAuditInfo()
		fixedRefresh     = domainfixtures.NewRefreshToken(func(refresh *domain.RefreshToken) {
			refresh.UserID = fixedUserID

			refresh.Audit = fixedAuditInfo
		})
		fixedClaims = corefixtures.NewJwtClaims(func(claims *core.JwtClaims) {
			claims.UserID = fixedUserID
		})

		fixedNow = time.Now().Truncate(time.Second)
	)

	tests := []struct {
		name    string
		input   Input
		prepare func(f *fields)
		check   func(t *testing.T, out Output, err error, f *fields)
	}{
		{
			name: "success path with existing redis session",
			input: Input{
				RefreshTokenString: validTokenStr,
			},
			prepare: func(f *fields) {
				f.postgres.GetRefreshTokenByHashFunc = func(ctx context.Context, hash string) (domain.RefreshToken, error) {
					return fixedRefresh, nil
				}
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, userID uuid.UUID) ([]core.Role, error) {
					return fixedRoles, nil
				}
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return nil
				}
				f.tokenGenerator.GenerateAndGetClaimsFunc = func(userID uuid.UUID) (core.AccessToken, core.JwtClaims, error) {
					return fixedAccessToken, fixedClaims, nil
				}
				f.postgres.RotateRefreshTokenFunc = func(ctx context.Context, oldHash string, newHash string, newJti core.Jti) error {
					return nil
				}
				f.redis.SetFunc = func(ctx context.Context, key string, value any, ttl time.Duration) error {
					return nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, fixedAccessToken, out.AccessToken)
				assert.Equal(t, fixedRefresh.ExpiresAt, out.RefreshTokenExpiresAt)
			},
		},
		{
			name: "success path without redis session",
			input: Input{
				RefreshTokenString: validTokenStr,
			},
			prepare: func(f *fields) {
				f.postgres.GetRefreshTokenByHashFunc = func(ctx context.Context, hash string) (domain.RefreshToken, error) {
					return fixedRefresh, nil
				}
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, userID uuid.UUID) ([]core.Role, error) {
					return fixedRoles, nil
				}
				f.redis.DeleteFunc = func(ctx context.Context, key string) error {
					return errs.ErrKeyNotFound
				}
				f.tokenGenerator.GenerateAndGetClaimsFunc = func(userID uuid.UUID) (core.AccessToken, core.JwtClaims, error) {
					return fixedAccessToken, fixedClaims, nil
				}
				f.postgres.RotateRefreshTokenFunc = func(ctx context.Context, oldHash string, newHash string, newJti core.Jti) error {
					return nil
				}
				f.redis.SetFunc = func(ctx context.Context, key string, value any, ttl time.Duration) error {
					return nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, fixedAccessToken, out.AccessToken)
				assert.Equal(t, fixedRefresh.ExpiresAt, out.RefreshTokenExpiresAt)
			},
		},
		{
			name: "error - invalid format token",
			input: Input{
				RefreshTokenString: "invalid token 123",
			},
			prepare: func(f *fields) {},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrInvalidArgument))
			},
		},
		{
			name: "error - refresh token not found",
			input: Input{
				RefreshTokenString: validTokenStr,
			},
			prepare: func(f *fields) {
				f.postgres.GetRefreshTokenByHashFunc = func(ctx context.Context, hash string) (domain.RefreshToken, error) {
					return domain.RefreshToken{}, errs.ErrNotFound
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrAuthorizationFailed))
				assert.Contains(t, err.Error(), "token is expired")
			},
		},
		{
			name: "error - refresh token is expired",
			input: Input{
				RefreshTokenString: validTokenStr,
			},
			prepare: func(f *fields) {
				f.postgres.GetRefreshTokenByHashFunc = func(ctx context.Context, hash string) (domain.RefreshToken, error) {
					return domainfixtures.NewRefreshToken(func(refresh *domain.RefreshToken) {
						refresh.ExpiresAt = fixedNow.Add(-time.Minute)
					}), nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrAuthorizationFailed))
				assert.Contains(t, err.Error(), "token is expired")
			},
		},
		{
			name: "error - session is compromised",
			input: Input{
				RefreshTokenString: validTokenStr,
			},
			prepare: func(f *fields) {
				f.postgres.GetRefreshTokenByHashFunc = func(ctx context.Context, hash string) (domain.RefreshToken, error) {
					return domainfixtures.NewRefreshToken(func(refresh *domain.RefreshToken) {
						refresh.ExpiresAt = fixedNow.Add(-time.Minute)
						refresh.Audit.Fingerprint = "other fingerprint"
					}), nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrSessionIsCompromised))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				postgres:       &MockPostgres{},
				redis:          &MockRedis{},
				tokenGenerator: &MockTokenGenerator{},
			}
			tt.prepare(f)

			ctx := fixedAuditInfo.WithContext(test.ContextWithZeroLogger())

			uc := New(f.postgres, f.redis, f.tokenGenerator)
			out, err := uc.Refresh(ctx, tt.input)

			tt.check(t, out, err, f)
		})
	}
}
