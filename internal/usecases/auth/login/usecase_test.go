package login

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
	"pinnAutomizer/internal/domain"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestUsecase_Login(t *testing.T) {
	type fields struct {
		postgres   *MockPostgres
		jwtService *MockJwtService
		hasher     *MockPasswordHasher
	}

	tests := []struct {
		name     string
		input    Input
		expected Output
		prepare  func(f fields)
		wantErr  bool
	}{
		{
			name:  "Valid",
			input: Input{Login: "admin", Password: "admin"},
			expected: Output{
				AccessToken: domain.Token{
					Value: "123",
				},
				RefreshToken: domain.Token{
					Value: "456",
				},
			},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					GetUserByLogin(mock.Anything, "admin").
					Return(domain.User{
						Login:        "admin",
						PasswordHash: "hash",
					}, nil).Once()

				f.hasher.EXPECT().
					CompareHashAndPassword("hash", "admin").
					Return(nil).Once()

				tokensPair := domain.TokensPair{
					AccessToken: domain.Token{
						Value:     "123",
						ExpiresAt: time.Time{},
					},
					RefreshToken: domain.Token{
						Value:     "456",
						ExpiresAt: time.Time{},
					},
				}

				authTokens, _ := domain.NewAuthToken(
					uuid.UUID{},
					tokensPair.AccessToken.Value,
					tokensPair.RefreshToken.Value,
				)

				f.jwtService.EXPECT().
					GenerateTokensPair(uuid.UUID{}).
					Return(tokensPair, nil).Once()

				f.postgres.EXPECT().
					Login(mock.Anything, authTokens).
					Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name:  "User not exists",
			input: Input{Login: "admin", Password: "admin"},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					GetUserByLogin(mock.Anything, "admin").
					Return(domain.User{}, pg_errors.ErrNotFound).Once()
			},
			wantErr: true,
		},
		{
			name:  "Small login",
			input: Input{Login: "a", Password: "admin"},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
		{
			name:  "Empty password",
			input: Input{Login: "a", Password: ""},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
		{
			name:  "Wrong password",
			input: Input{Login: "admin", Password: "admin"},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					GetUserByLogin(mock.Anything, "admin").
					Return(domain.User{
						Login:        "admin",
						PasswordHash: "hash",
					}, nil).Once()

				f.hasher.EXPECT().
					CompareHashAndPassword("hash", "admin").
					Return(bcrypt.ErrMismatchedHashAndPassword).Once()
			},
			wantErr: true,
		},
		{
			name:  "Tokens saving error",
			input: Input{Login: "admin", Password: "admin"},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					GetUserByLogin(mock.Anything, "admin").
					Return(domain.User{
						Login:        "admin",
						PasswordHash: "hash",
					}, nil).Once()

				f.hasher.EXPECT().
					CompareHashAndPassword("hash", "admin").
					Return(nil).Once()

				tokensPair := domain.TokensPair{
					AccessToken: domain.Token{
						Value:     "123",
						ExpiresAt: time.Time{},
					},
					RefreshToken: domain.Token{
						Value:     "456",
						ExpiresAt: time.Time{},
					},
				}

				authTokens, _ := domain.NewAuthToken(
					uuid.UUID{},
					tokensPair.AccessToken.Value,
					tokensPair.RefreshToken.Value,
				)

				f.jwtService.EXPECT().
					GenerateTokensPair(uuid.UUID{}).
					Return(tokensPair, nil).Once()

				f.postgres.EXPECT().
					Login(mock.Anything, authTokens).
					Return(pg_errors.ErrNotFound).Once()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				postgres:   NewMockPostgres(t),
				jwtService: NewMockJwtService(t),
				hasher:     NewMockPasswordHasher(t),
			}
			tt.prepare(f)

			uc := New(f.postgres, f.jwtService, f.hasher, zerolog.Logger{})
			actual, err := uc.Login(context.Background(), tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}
