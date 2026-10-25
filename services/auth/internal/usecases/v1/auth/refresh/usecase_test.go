package authRefresh

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func refreshDependencies(fixture refreshFixture) (*MockPostgres, *MockRedis) {
	postgres := &MockPostgres{
		GetRefreshTokenByHashFunc: func(context.Context, string) (domain.RefreshToken, error) {
			return fixture.token, nil
		},
		GetRolesByUserIDFunc: func(context.Context, uuid.UUID) ([]core.Role, error) {
			return fixture.roles, nil
		},
		RotateRefreshTokenFunc: func(context.Context, string, string, core.Jti) (core.Changed, error) {
			return true, nil
		},
	}
	redis := &MockRedis{
		SetFunc: func(context.Context, string, any, time.Duration) error {
			return nil
		},
		DeleteFunc: func(context.Context, string) error {
			return nil
		},
	}
	return postgres, redis
}

func TestRefreshSucceedsWithPresentOrMissingOldSession(t *testing.T) {
	t.Parallel()
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing session", true: "missing session"}[missing], func(t *testing.T) {
			t.Parallel()
			fixture := newRefreshFixture(t)
			postgres, redis := refreshDependencies(fixture)
			if missing {
				redis.DeleteFunc = func(context.Context, string) error {
					return errs.ErrKeyNotFound
				}
			}
			out, err := New(postgres, redis, fixture.generator).Refresh(fixture.token.Audit.WithContext(t.Context()), fixture.input)
			require.NoError(t, err)
			claims, err := fixture.generator.GetClaims(out.AccessToken)
			require.NoError(t, err)
			require.Equal(t, fixture.token.UserID, claims.UserID)
			require.Equal(t, fixture.token.ExpiresAt, out.RefreshTokenExpiresAt)
			require.NotEqual(t, fixture.input.RefreshTokenString, out.RefreshTokenString)
			require.NoError(t, (Input{RefreshTokenString: out.RefreshTokenString}).Validate())
		})
	}
}

func TestRefreshRejectsInvalidOrUnknownToken(t *testing.T) {
	t.Parallel()
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "not found", true: "invalid input"}[invalid], func(t *testing.T) {
			t.Parallel()
			fixture := newRefreshFixture(t)
			postgres, redis := refreshDependencies(fixture)
			postgres.GetRefreshTokenByHashFunc = func(context.Context, string) (domain.RefreshToken, error) {
				return domain.RefreshToken{}, errs.ErrNotFound
			}
			want := errs.ErrAuthorizationFailed
			if invalid {
				fixture.input.RefreshTokenString = "invalid token 123"
				want = errs.ErrInvalidArgument
			}
			out, err := New(postgres, redis, fixture.generator).Refresh(fixture.token.Audit.WithContext(t.Context()), fixture.input)
			require.ErrorIs(t, err, want)
			require.Empty(t, out)
			require.Empty(t, redis.SetCalls())
			require.Empty(t, redis.DeleteCalls())
			if invalid {
				require.Empty(t, postgres.GetRefreshTokenByHashCalls())
			}
		})
	}
}

func TestRefreshRejectsExpiredTokenOrWrongFingerprint(t *testing.T) {
	t.Parallel()
	for _, expired := range []bool{false, true} {
		t.Run(map[bool]string{false: "compromised even when expired", true: "expired"}[expired], func(t *testing.T) {
			t.Parallel()
			fixture := newRefreshFixture(t)
			audit := fixture.token.Audit
			snapshot := domain.RefreshTokenSnapshot(fixture.token)
			want := errs.ErrSessionIsCompromised
			snapshot.CreatedAt = time.Now().Add(-2 * time.Hour)
			snapshot.ExpiresAt = time.Now().Add(-time.Hour)
			if expired {
				want = errs.ErrAuthorizationFailed
			} else {
				fingerprint, err := core.NewFingerprint(strings.Repeat("b", 64))
				require.NoError(t, err)
				snapshot.Audit = core.NewAuditInfo(fingerprint, audit.IP, audit.Agent)
			}
			var err error
			fixture.token, err = domain.RestoreRefreshToken(snapshot)
			require.NoError(t, err)
			postgres, redis := refreshDependencies(fixture)
			out, err := New(postgres, redis, fixture.generator).Refresh(audit.WithContext(t.Context()), fixture.input)
			require.ErrorIs(t, err, want)
			require.Empty(t, out)
			require.Empty(t, redis.SetCalls())
			require.Empty(t, redis.DeleteCalls())
			if expired {
				require.ErrorContains(t, err, "token is expired")
			}
		})
	}
}
