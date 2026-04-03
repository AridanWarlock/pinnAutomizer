package update_task_status_after_train

import (
	"context"
	"pinnAutomizer/internal/adapter/postgres/pg_errors"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/test"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
)

func TestUsecase_UpdateTaskStatusAfterTrain(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
	}

	tests := []struct {
		name    string
		input   Input
		prepare func(f fields, i Input)
		wantErr bool
	}{
		{
			name: "valid path",
			input: Input{
				ID: uuid.New(),
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					UpdateTaskStatusByID(
						mock.Anything,
						i.ID,
						string(domain.TaskStatusDone),
						string(domain.TaskStatusTraining),
					).
					Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "task not exist",
			input: Input{
				ID: uuid.New(),
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					UpdateTaskStatusByID(
						mock.Anything,
						i.ID,
						string(domain.TaskStatusDone),
						string(domain.TaskStatusTraining),
					).
					Return(pg_errors.ErrNotFound).Once()
			},
			wantErr: true,
		},
		{
			name:  "invalid input",
			input: Input{},
			prepare: func(f fields, i Input) {
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				postgres: NewMockPostgres(t),
			}
			tt.prepare(f, tt.input)

			uc := New(f.postgres, zerolog.Logger{})
			err := uc.UpdateTaskStatusAfterTrain(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)
		})
	}
}
