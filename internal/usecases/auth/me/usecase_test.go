package me

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/test"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUsecase_Me(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
	}

	tests := []struct {
		name     string
		input    Input
		expected Output
		prepare  func(f fields)
		wantErr  bool
	}{
		{
			name:     "valid path",
			input:    Input{ID: uuid.Max},
			expected: Output{ID: uuid.Max, Login: "admin"},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					GetUserByID(mock.Anything, uuid.Max).
					Return(domain.User{
						ID:    uuid.Max,
						Login: "admin",
					}, nil).Once()
			},
			wantErr: false,
		},
		{
			name:  "user not exist",
			input: Input{ID: uuid.Max},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					GetUserByID(mock.Anything, uuid.Max).
					Return(domain.User{}, pg_errors.ErrNotFound).
					Once()
			},
			wantErr: true,
		},
		{
			name:  "invalid input",
			input: Input{ID: uuid.UUID{}},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				postgres: NewMockPostgres(t),
			}

			tt.prepare(f)

			uc := New(f.postgres, zerolog.Logger{})
			actual, err := uc.Me(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
