package create_task

import (
	"context"
	"encoding/json"
	"math"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/test"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
)

func TestUsecase_CreateTask(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
	}

	tests := []struct {
		name     string
		input    Input
		expected Output
		prepare  func(f fields, i Input)
		wantErr  bool
	}{
		{
			name: "valid path",
			input: Input{
				Name:         "name",
				Description:  "desc",
				Constants:    map[string]any{},
				UserID:       uuid.New(),
				EquationType: domain.EquationTypeHeat,
			},
			expected: Output{
				Task: domain.Task{
					Name:        "name",
					Description: "desc",
					Status:      domain.TaskStatusCreated,
					Constants:   map[string]any{},
					ResultsPath: "",
				},
				Equation: domain.Equation{
					Type: domain.EquationTypeHeat,
				},
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					GetEquationByType(mock.Anything, domain.EquationTypeHeat).
					Return(domain.Equation{
						ID:   uuid.New(),
						Type: domain.EquationTypeHeat,
					}, nil).Once()

				f.postgres.EXPECT().
					CreateTask(mock.Anything, mock.MatchedBy(func(task domain.Task) bool {
						return task.UserID == i.UserID
					})).
					RunAndReturn(func(ctx context.Context, task domain.Task) (domain.Task, error) {
						return task, nil
					})

				f.postgres.EXPECT().
					PublishEvent(mock.Anything, mock.MatchedBy(func(event domain.Event) bool {
						if err := json.Unmarshal(event.Data, &domain.TrainMessage{}); err != nil {
							return false
						}

						return event.Topic == "to-train"
					})).Return(nil).Once()

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					}).Once()
			},
			wantErr: false,
		},
		{
			name: "publish event error",
			input: Input{
				Name:         "name",
				Description:  "desc",
				Constants:    map[string]any{},
				UserID:       uuid.New(),
				EquationType: domain.EquationTypeHeat,
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					GetEquationByType(mock.Anything, domain.EquationTypeHeat).
					Return(domain.Equation{
						ID:   uuid.New(),
						Type: domain.EquationTypeHeat,
					}, nil).Once()

				f.postgres.EXPECT().
					CreateTask(mock.Anything, mock.MatchedBy(func(task domain.Task) bool {
						return task.UserID == i.UserID
					})).
					RunAndReturn(func(ctx context.Context, task domain.Task) (domain.Task, error) {
						return task, nil
					})

				f.postgres.EXPECT().
					PublishEvent(mock.Anything, mock.Anything).
					Return(pgx.ErrTxClosed).Once()

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					}).Once()
			},
			wantErr: true,
		},
		{
			name: "json marshal error",
			input: Input{
				Name:        "name",
				Description: "desc",
				Constants: map[string]any{
					"invalid": math.NaN(),
				},
				UserID:       uuid.New(),
				EquationType: domain.EquationTypeHeat,
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					GetEquationByType(mock.Anything, domain.EquationTypeHeat).
					Return(domain.Equation{
						ID:   uuid.New(),
						Type: domain.EquationTypeHeat,
					}, nil).Once()

				f.postgres.EXPECT().
					CreateTask(mock.Anything, mock.MatchedBy(func(task domain.Task) bool {
						return task.UserID == i.UserID
					})).
					RunAndReturn(func(ctx context.Context, task domain.Task) (domain.Task, error) {
						return task, nil
					})

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					}).Once()
			},
			wantErr: true,
		},
		{
			name: "create task error",
			input: Input{
				Name:         "name",
				Description:  "desc",
				Constants:    map[string]any{},
				UserID:       uuid.New(),
				EquationType: domain.EquationTypeHeat,
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					GetEquationByType(mock.Anything, domain.EquationTypeHeat).
					Return(domain.Equation{
						ID:   uuid.New(),
						Type: domain.EquationTypeHeat,
					}, nil).Once()

				f.postgres.EXPECT().
					CreateTask(mock.Anything, mock.Anything).
					Return(domain.Task{}, pgx.ErrTxClosed).Once()

				f.postgres.EXPECT().
					Wrap(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					}).Once()
			},
			wantErr: true,
		},
		{
			name: "get equation by type error",
			input: Input{
				Name:         "name",
				Description:  "desc",
				Constants:    map[string]any{},
				UserID:       uuid.New(),
				EquationType: domain.EquationTypeHeat,
			},
			prepare: func(f fields, i Input) {
				f.postgres.EXPECT().
					GetEquationByType(mock.Anything, domain.EquationTypeHeat).
					Return(domain.Equation{}, pgx.ErrTxClosed).Once()
			},
			wantErr: true,
		},
		{
			name: "invalid input",
			input: Input{
				Name:         "name",
				Description:  "desc",
				Constants:    map[string]any{},
				UserID:       uuid.New(),
				EquationType: "some not existing equation",
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
			actual, err := uc.CreateTask(context.Background(), tt.input)

			test.AssertErr(t, err, tt.wantErr)

			diff := cmp.Diff(tt.expected, actual, cmpopts.IgnoreFields(Output{},
				"Task.ID", "Task.CreatedAt", "Task.TrainingDataPath", "Task.UserID", "Task.EquationID",
				"Equation.ID",
			))
			if diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
