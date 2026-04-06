package auth_refresh

import (
	"context"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pg_errors"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUsecase_Refresh(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
		jwt      *MockJwtService
	}

	tests := []struct {
		name     string
		input    Input
		expected Output
		prepare  func(f fields)
		wantErr  bool
	}{
		{
			name: "valid path",
			input: Input{
				RefreshTokenString: "valid_refresh",
			},
			expected: Output{AccessToken: domain.AccessToken{
				Value:     "valid_access",
				ExpiresAt: time.Time{}.Add(time.Hour),
			}},
			prepare: func(f fields) {
				f.jwt.EXPECT().
					ValidateRefreshToken(mock.Anything, "valid_refresh").
					Return(uuid.Max, nil).
					Once()

				f.jwt.EXPECT().
					GenerateAccessToken(uuid.Max).
					Return(domain.AccessToken{
						Value:     "valid_access",
						ExpiresAt: time.Time{}.Add(time.Hour),
					}, nil).Once()

				f.postgres.EXPECT().
					Refresh(mock.Anything, uuid.Max, "valid_access").
					Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "invalid input",
			input: Input{
				RefreshTokenString: "",
			},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
		{
			name: "postgres refresh error",
			input: Input{
				RefreshTokenString: "valid_refresh",
			},
			prepare: func(f fields) {
				f.jwt.EXPECT().
					ValidateRefreshToken(mock.Anything, "valid_refresh").
					Return(uuid.Max, nil).
					Once()

				f.jwt.EXPECT().
					GenerateAccessToken(uuid.Max).
					Return(domain.AccessToken{
						Value:     "valid_access",
						ExpiresAt: time.Time{}.Add(time.Hour),
					}, nil).Once()

				f.postgres.EXPECT().
					Refresh(mock.Anything, uuid.Max, "valid_access").
					Return(pg_errors.ErrUpdateRowsAffectedCount).Once()
			},
			wantErr: true,
		},
		{
			name: "generate token error",
			input: Input{
				RefreshTokenString: "valid_refresh",
			},
			prepare: func(f fields) {
				f.jwt.EXPECT().
					ValidateRefreshToken(mock.Anything, "valid_refresh").
					Return(uuid.Max, nil).
					Once()

				f.jwt.EXPECT().
					GenerateAccessToken(uuid.Max).
					Return(domain.AccessToken{}, jwt.ErrSignatureInvalid).
					Once()
			},
			wantErr: true,
		},
		{
			name: "validate token error",
			input: Input{
				RefreshTokenString: "valid_refresh",
			},
			prepare: func(f fields) {
				f.jwt.EXPECT().
					ValidateRefreshToken(mock.Anything, "valid_refresh").
					Return(uuid.UUID{}, jwt.ErrTokenExpired).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				postgres: NewMockPostgres(t),
				jwt:      NewMockJwtService(t),
			}
			tt.prepare(f)

			uc := New(f.postgres, f.jwt, zerolog.Logger{})
			actual, err := uc.Refresh(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
