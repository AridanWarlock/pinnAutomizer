package authRefresh

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUsecase_Refresh(t *testing.T) {
	type fields struct {
		postgres        *MockPostgres
		accessGenerator *MockAccessTokenGenerator
	}

	var (
		fixedID          = uuid.New()
		fixedUserID      = uuid.New()
		fixedTokenRaw    = "random_string"
		validTokenStr    = fmt.Sprintf("%s.%s", fixedID.String(), fixedTokenRaw)
		fixedHash        = sha256.Sum256([]byte(fixedTokenRaw))
		fixedFingerprint = fixtures.NewFingerprint()
		fixedAccessToken = fixtures.NewAccessToken()
		fixedNow         = time.Now()
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
				RefreshTokenString: validTokenStr,
				Fingerprint:        fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserSessionByIdFunc = func(ctx context.Context, id uuid.UUID) (domain.UserSession, error) {
					return fixtures.NewUserSession(func(s *domain.UserSession) {
						s.ID = fixedID
						s.UserID = fixedUserID
						s.TokenSha256 = fixedHash[:]
						s.CreatedAt = fixedNow
						s.ExpiresAt = fixedNow.Add(time.Hour)
					}), nil
				}
				f.postgres.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.User, error) {
					return fixtures.NewUser(func(u *domain.User) {
						u.ID = fixedUserID
						u.Login = "admin"
					}), nil
				}
				f.postgres.GetRolesByUserIDFunc = func(ctx context.Context, id uuid.UUID) ([]domain.Role, error) {
					return []domain.Role{}, nil
				}
				f.accessGenerator.GenerateFunc = func(
					user domain.User,
					roles []domain.Role,
					fingerprint domain.Fingerprint,
				) (domain.AccessToken, error) {
					return fixedAccessToken, nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, fixedAccessToken, out.AccessToken)
			},
		},
		{
			name: "error - invalid format token (no dot)",
			input: Input{
				RefreshTokenString: "invalid-format-token",
				Fingerprint:        fixedFingerprint,
			},
			prepare: func(f *fields) {},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrInvalidArgument))
			},
		},
		{
			name: "error - invalid fingerprint",
			input: Input{
				RefreshTokenString: validTokenStr,
				Fingerprint:        []byte("bad fingerprint"),
			},
			prepare: func(f *fields) {},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrInvalidArgument))
			},
		},
		{
			name: "error - session not found",
			input: Input{
				RefreshTokenString: validTokenStr,
				Fingerprint:        fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserSessionByIdFunc = func(ctx context.Context, id uuid.UUID) (domain.UserSession, error) {
					return domain.UserSession{}, errs.ErrNotFound
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrNotFound))
			},
		},
		{
			name: "error - session compromised (hash mismatch)",
			input: Input{
				RefreshTokenString: validTokenStr,
				Fingerprint:        fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserSessionByIdFunc = func(ctx context.Context, id uuid.UUID) (domain.UserSession, error) {
					return fixtures.NewUserSession(func(s *domain.UserSession) {
						s.ID = fixedID
						s.TokenSha256 = []byte("wrong-hash")
						s.CreatedAt = fixedNow
						s.ExpiresAt = fixedNow.Add(time.Hour)
						s.Fingerprint = fixedFingerprint
					}), nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrSessionIsCompromised))
				assert.Len(t, f.postgres.GetUserByIDCalls(), 0)
			},
		},
		{
			name: "error - token expired",
			input: Input{
				RefreshTokenString: validTokenStr,
				Fingerprint:        fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserSessionByIdFunc = func(ctx context.Context, id uuid.UUID) (domain.UserSession, error) {
					return fixtures.NewUserSession(func(s *domain.UserSession) {
						s.ID = fixedID
						s.TokenSha256 = fixedHash[:]
						s.CreatedAt = fixedNow
						s.ExpiresAt = fixedNow.Add(-time.Hour)
						s.Fingerprint = fixedFingerprint
					}), nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrAuthorizationFailed))
			},
		},
		{
			name: "error - user not found",
			input: Input{
				RefreshTokenString: validTokenStr,
				Fingerprint:        fixedFingerprint,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserSessionByIdFunc = func(ctx context.Context, id uuid.UUID) (domain.UserSession, error) {
					return fixtures.NewUserSession(func(s *domain.UserSession) {
						s.TokenSha256 = fixedHash[:]
					}), nil
				}
				f.postgres.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.User, error) {
					return domain.User{}, errs.ErrNotFound
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, errs.ErrNotFound))
				assert.Len(t, f.postgres.GetRolesByUserIDCalls(), 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				postgres:        &MockPostgres{},
				accessGenerator: &MockAccessTokenGenerator{},
			}
			tt.prepare(f)

			uc := New(f.postgres, f.accessGenerator)
			out, err := uc.Refresh(testCtx, tt.input)

			tt.check(t, out, err, f)
		})
	}
}
