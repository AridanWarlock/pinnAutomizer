package authLogin

import (
	"context"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"

	"github.com/golang-jwt/jwt/v5"
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
			name:  "valid path",
			input: Input{Login: "admin", Password: "12345"},
			expected: Output{
				AccessToken: domain.AccessToken{
					Value: "123",
				},
				RefreshToken: domain.AccessToken{
					Value: "456",
				},
			},
			prepare: func(f fields) {
				user := domain.User{
					ID:           uuid.New(),
					Login:        "admin",
					PasswordHash: "hash",
				}

				f.postgres.GetUserByLoginFunc = func(ctx context.Context, login string) (domain.User, error) {
					return user, nil
				}

				f.postgres.EXPECT().
					GetUserByLogin(mock.Anything, "admin").
					Return(user, nil).Once()

				f.hasher.EXPECT().
					CompareHashAndPassword(user.PasswordHash, "12345").
					Return(nil).Once()

				tokensPair := domain.TokensPair{
					AccessToken: domain.AccessToken{
						Value:     "123",
						ExpiresAt: time.Time{},
					},
					RefreshToken: domain.AccessToken{
						Value:     "456",
						ExpiresAt: time.Time{},
					},
				}

				authTokens, _ := domain.NewAuthToken(
					user.ID,
					tokensPair.AccessToken.Value,
					tokensPair.RefreshToken.Value,
				)

				f.jwtService.EXPECT().
					GenerateTokensPair(user.ID).
					Return(tokensPair, nil).Once()

				f.postgres.EXPECT().
					Login(mock.Anything, authTokens).
					Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name:  "generate tokens failed",
			input: Input{Login: "admin", Password: "admin"},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					GetUserByLogin(mock.Anything, "admin").
					Return(domain.User{
						ID:           uuid.New(),
						Login:        "admin",
						PasswordHash: "hash",
					}, nil).Once()

				f.hasher.EXPECT().
					CompareHashAndPassword("hash", "admin").
					Return(nil).Once()

				f.jwtService.EXPECT().
					GenerateTokensPair(mock.Anything).
					Return(domain.TokensPair{}, jwt.ErrHashUnavailable).Once()
			},
			wantErr: true,
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
						ID:           uuid.New(),
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
			name:  "tokens saving error",
			input: Input{Login: "admin", Password: "admin"},
			prepare: func(f fields) {
				user := domain.User{
					ID:           uuid.New(),
					Login:        "admin",
					PasswordHash: "hash",
				}

				f.postgres.EXPECT().
					GetUserByLogin(mock.Anything, "admin").
					Return(user, nil).Once()

				f.hasher.EXPECT().
					CompareHashAndPassword(user.PasswordHash, "admin").
					Return(nil).Once()

				tokensPair := domain.TokensPair{
					AccessToken: domain.AccessToken{
						Value:     "123",
						ExpiresAt: time.Time{},
					},
					RefreshToken: domain.AccessToken{
						Value:     "456",
						ExpiresAt: time.Time{},
					},
				}

				authTokens, _ := domain.NewAuthToken(
					user.ID,
					tokensPair.AccessToken.Value,
					tokensPair.RefreshToken.Value,
				)

				f.jwtService.EXPECT().
					GenerateTokensPair(mock.Anything).
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
