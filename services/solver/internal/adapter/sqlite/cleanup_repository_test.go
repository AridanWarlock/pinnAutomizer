package sqlite

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestConcurrentCleanupMarkerSurvivesReopenAndLeavesOutboxIntact(t *testing.T) {
	cfg := config(t)
	repo := open(t, cfg)
	a := fixtures.NewAttempt(t)
	_, _, err := repo.RegisterAttempt(t.Context(), a)
	require.NoError(t, err)
	_, err = repo.RecordContainer(t.Context(), a.RunID, "id")
	require.NoError(t, err)
	code := 0
	_, err = repo.FinishAttempt(t.Context(), a.RunID, &code, nil, time.Now(), event(t, a))
	require.NoError(t, err)
	pending, err := repo.GetRecoverableAttempts(t.Context(), uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.True(t, pending[0].NeedsCleanup())

	type result struct {
		changed core.Changed
		err     error
	}
	results := make(chan result, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Go(func() {
			changed, err := repo.MarkCleaned(t.Context(), a.RunID, time.Now())
			results <- result{changed: changed, err: err}
		})
	}
	workers.Wait()
	close(results)
	changes := 0
	for r := range results {
		require.NoError(t, r.err)
		if r.changed {
			changes++
		}
	}
	require.Equal(t, 1, changes, "only the first confirmation changes the durable marker")
	stored, err := repo.GetAttempt(t.Context(), a.RunID)
	require.NoError(t, err)
	require.NotNil(t, stored.CleanedAt)
	require.NoError(t, repo.Close())
	repo = open(t, cfg)
	restored, err := repo.GetAttempt(t.Context(), a.RunID)
	require.NoError(t, err)
	require.Equal(t, stored, restored)
	pending, err = repo.GetRecoverableAttempts(t.Context(), uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, pending)
	events, err := repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, events, 1)
}
