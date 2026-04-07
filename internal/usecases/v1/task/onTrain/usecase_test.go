package tasksOnTrain

import (
	"context"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
)

func TestUsecase_UpdateTaskStatusOnTrain(t *testing.T) {
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
						string(domain.TaskStatusTraining),
						string(domain.TaskStatusCreated),
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
						string(domain.TaskStatusTraining),
						string(domain.TaskStatusCreated),
					).
					Return(errs.ErrNotFound).Once()
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

			uc := New(f.postgres)
			err := uc.UpdateTaskOnTrain(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)
		})
	}
}
