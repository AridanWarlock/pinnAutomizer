package tasksCleanup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestCleanupPersistsOnlyAfterSuccessfulRemoval(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"unfinished", "finished", "already cleaned", "no container", "remove failed", "save failed", "read failed"} {
		t.Run(scenario, func(t *testing.T) {
			a := fixtures.NewAttempt(t, fixtures.WithContainer("id"))
			if scenario == "no container" {
				a = fixtures.NewAttempt(t)
			}
			if scenario != "unfinished" {
				code := 0
				_, err := a.Finish(&code, nil, time.Now())
				require.NoError(t, err)
			}
			if scenario == "already cleaned" {
				_, err := a.MarkCleaned(time.Now())
				require.NoError(t, err)
			}
			failure := errors.New("injected failure")
			removed := false
			sqlite := &MockSQLite{
				GetAttemptFunc: func(context.Context, uuid.UUID) (domain.Attempt, error) {
					if scenario == "read failed" {
						return domain.Attempt{}, failure
					}
					return a, nil
				},
				MarkCleanedFunc: func(_ context.Context, id uuid.UUID, _ time.Time) (core.Changed, error) {
					require.True(t, removed, "journal marker must follow physical cleanup")
					require.Equal(t, a.RunID, id)
					if scenario == "save failed" {
						return false, failure
					}
					return true, nil
				},
			}
			runner := &MockPinnRunner{
				CleanupAttemptFunc: func(_ context.Context, actual domain.Attempt) error {
					require.Equal(t, a, actual)
					if scenario == "remove failed" {
						return failure
					}
					removed = true
					return nil
				},
			}
			out, err := New(sqlite, runner).CleanupTask(t.Context(), Input{RunID: a.RunID})
			switch scenario {
			case "unfinished":
				require.ErrorIs(t, err, errs.ErrConflict)
			case "remove failed", "save failed", "read failed":
				require.ErrorIs(t, err, failure)
			default:
				require.NoError(t, err)
			}
			require.Equal(t, core.Changed(scenario == "finished"), out.Cleaned)
			shouldSave := scenario == "finished" || scenario == "save failed"
			require.Equal(t, shouldSave, len(sqlite.MarkCleanedCalls()) == 1)
			shouldRemove := shouldSave || scenario == "remove failed"
			require.Equal(t, shouldRemove, len(runner.CleanupAttemptCalls()) == 1)
		})
	}
}

func TestCleanupRejectsInvalidInputBeforeReadingJournal(t *testing.T) {
	t.Parallel()
	sqlite := &MockSQLite{}
	_, err := New(sqlite, &MockPinnRunner{}).CleanupTask(t.Context(), Input{})
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
	require.Empty(t, sqlite.GetAttemptCalls())
}
