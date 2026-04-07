package tasksSolve

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
)

func TestUsecase_SolveTask(t *testing.T) {
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
				TaskID:    uuid.New(),
				UserID:    uuid.New(),
				Constants: map[string]any{},
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().GetTaskByIDAndUserID(mock.Anything, i.TaskID, i.UserID).Return(domain.Task{
					ID:          i.TaskID,
					Name:        "some",
					Description: "123",
					Status:      domain.TaskStatusDone,
					Constants:   map[string]any{},
					ResultsPath: "some/path",
					UserID:      i.UserID,
					EquationID:  uuid.New(),
					CreatedAt:   time.Now(),
				}, nil).Once()

				f.postgres.EXPECT().PublishEvent(mock.Anything, mock.MatchedBy(func(event domain.Event) bool {
					if err := json.Unmarshal(event.Data, &domain.SolveTaskMessage{}); err != nil {
						return false
					}

					return event.Topic == "to-solve"
				})).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "publish event failed",
			input: Input{
				TaskID:    uuid.New(),
				UserID:    uuid.New(),
				Constants: map[string]any{},
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().GetTaskByIDAndUserID(mock.Anything, i.TaskID, i.UserID).Return(domain.Task{
					ID:          i.TaskID,
					Name:        "some",
					Description: "123",
					Status:      domain.TaskStatusDone,
					Constants:   map[string]any{},
					ResultsPath: "some/path",
					UserID:      i.UserID,
					EquationID:  uuid.New(),
					CreatedAt:   time.Now(),
				}, nil).Once()

				f.postgres.EXPECT().
					PublishEvent(mock.Anything, mock.Anything).
					Return(pgx.ErrTxClosed).Once()
			},
			wantErr: true,
		},
		{
			name: "task not trained",
			input: Input{
				TaskID:    uuid.New(),
				UserID:    uuid.New(),
				Constants: map[string]any{},
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().GetTaskByIDAndUserID(mock.Anything, i.TaskID, i.UserID).Return(domain.Task{
					ID:          i.TaskID,
					Name:        "some",
					Description: "123",
					Status:      domain.TaskStatusTraining,
					Constants:   map[string]any{},
					ResultsPath: "",
					UserID:      i.UserID,
					EquationID:  uuid.New(),
					CreatedAt:   time.Now(),
				}, nil).Once()
			},
			wantErr: true,
		},
		{
			name: "task not exists",
			input: Input{
				TaskID:    uuid.New(),
				UserID:    uuid.New(),
				Constants: map[string]any{},
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					GetTaskByIDAndUserID(mock.Anything, i.TaskID, i.UserID).
					Return(domain.Task{}, errs.ErrNotFound).Once()
			},
			wantErr: true,
		},
		{
			name: "not valid json data",
			input: Input{
				TaskID: uuid.New(),
				UserID: uuid.New(),
				Constants: map[string]any{
					"not parsed to json": math.Inf(1),
				},
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().GetTaskByIDAndUserID(mock.Anything, i.TaskID, i.UserID).Return(domain.Task{
					ID:          i.TaskID,
					Name:        "some",
					Description: "123",
					Status:      domain.TaskStatusDone,
					Constants: map[string]any{
						"not parsed to json": math.Inf(1),
					},
					ResultsPath: "",
					UserID:      i.UserID,
					EquationID:  uuid.New(),
					CreatedAt:   time.Now(),
				}, nil).Once()
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
			err := uc.SolveTask(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)
		})
	}
}
