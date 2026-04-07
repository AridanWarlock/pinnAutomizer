package tasksGet

import (
	"context"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUsecase_GetTasks(t *testing.T) {
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
				IDs:    []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
				UserID: uuid.New(),
			},
			prepare: func(f fields, i Input) {
				eqIDs := []uuid.UUID{
					uuid.New(), uuid.New(),
				}
				tasks := []domain.Task{
					{
						EquationID: eqIDs[0],
					},
					{
						EquationID: eqIDs[1],
					},
					{
						EquationID: eqIDs[0],
					},
				}

				f.postgres.EXPECT().
					GetTasksByIDs(mock.Anything, i.IDs, i.UserID).
					Return(tasks, nil).Once()

				f.postgres.EXPECT().
					GetEquationsByIDs(mock.Anything, eqIDs).
					Return([]domain.Equation{
						{
							ID: eqIDs[0],
						},
						{
							ID: eqIDs[1],
						},
					}, nil).Once()
			},
			wantErr: false,
		},
		{
			name: "not existing tasks",
			input: Input{
				IDs:    []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
				UserID: uuid.New(),
			},
			prepare: func(f fields, i Input) {
				tasks := make([]domain.Task, 2)

				f.postgres.EXPECT().
					GetTasksByIDs(mock.Anything, i.IDs, i.UserID).
					Return(tasks, nil).Once()
			},
			wantErr: true,
		},
		{
			name: "get equations failed",
			input: Input{
				IDs:    []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
				UserID: uuid.New(),
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					GetTasksByIDs(mock.Anything, i.IDs, i.UserID).
					Return(make([]domain.Task, 3), nil).Once()

				f.postgres.EXPECT().
					GetEquationsByIDs(mock.Anything, mock.Anything).
					Return(nil, pgx.ErrTxClosed).Once()
			},
			wantErr: true,
		},
		{
			name: "get tasks failed",
			input: Input{
				IDs:    []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
				UserID: uuid.New(),
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					GetTasksByIDs(mock.Anything, i.IDs, i.UserID).
					Return(nil, pgx.ErrTxClosed).Once()
			},
			wantErr: true,
		},
		{
			name: "empty ids",
			input: Input{
				IDs:    []uuid.UUID{},
				UserID: uuid.New(),
			},
			prepare: func(f fields, i Input) {
			},
			wantErr: true,
		},
		{
			name: "too many ids",
			input: Input{
				IDs:    make([]uuid.UUID, 100),
				UserID: uuid.New(),
			},
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
			actual, err := uc.GetTasks(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)
			if err != nil {
				return
			}

			assert.Equal(t, len(tt.input.IDs), len(actual.TasksToEquation))
		})
	}
}
