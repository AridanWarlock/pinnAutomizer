package auth_logout

import (
	"context"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pg_errors"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
)

func TestUsecase_Logout(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
	}

	tests := []struct {
		name    string
		input   Input
		prepare func(f fields)
		wantErr bool
	}{
		{
			name:  "valid path",
			input: Input{ID: uuid.Max},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					Logout(mock.Anything, uuid.Max).
					Return(nil).
					Once()
			},
			wantErr: false,
		},
		{
			name:  "invalid input",
			input: Input{ID: uuid.UUID{}},
			prepare: func(f fields) {
			},
			wantErr: true,
		},
		{
			name:  "user not found",
			input: Input{ID: uuid.Max},
			prepare: func(f fields) {
				f.postgres.EXPECT().
					Logout(mock.Anything, uuid.Max).
					Return(pg_errors.ErrNotFound).
					Once()
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
			err := uc.Logout(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)
		})
	}
}
