package sqlite

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestConcurrentStartAdmissionKeepsActiveLimitAndReleasesOnCompletion(t *testing.T) {
	repo := open(t, config(t))
	attempts := []domain.Attempt{fixtures.NewAttempt(t), fixtures.NewAttempt(t)}
	for _, a := range attempts {
		_, _, err := repo.RegisterAttempt(t.Context(), a)
		require.NoError(t, err)
		_, err = repo.RecordContainer(t.Context(), a.RunID, a.RunID.String())
		require.NoError(t, err)
	}
	type result struct {
		id      int
		changed core.Changed
		err     error
	}
	results := make(chan result, len(attempts))
	events := []domain.Event{fixtures.NewEvent(t, attempts[0]), fixtures.NewEvent(t, attempts[1])}
	var workers sync.WaitGroup
	for i, a := range attempts {
		workers.Go(func() {
			changed, err := repo.RequestStart(t.Context(), a.RunID, time.Now(), time.Minute, events[i], limit.MustNew(1))
			results <- result{id: i, changed: changed, err: err}
		})
	}
	workers.Wait()
	close(results)
	winner, waiting := -1, -1
	for result := range results {
		require.NoError(t, result.err)
		if result.changed {
			winner = result.id
		} else {
			waiting = result.id
		}
	}
	require.NotEqual(t, -1, winner)
	require.NotEqual(t, -1, waiting, "the other attempt must remain eligible for a later sweep")
	pending, err := repo.GetAttempt(t.Context(), attempts[waiting].RunID)
	require.NoError(t, err)
	require.Equal(t, domain.AttemptContainerCreated, pending.State)
	require.Nil(t, pending.Deadline)
	messages, err := repo.GetPendingEvents(t.Context(), limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, messages, 1, "capacity rejection must not publish a false start event")
	code := 0
	_, err = repo.FinishAttempt(t.Context(), attempts[winner].RunID, &code, nil, time.Now(), event(t, attempts[winner]))
	require.NoError(t, err)
	changed, err := repo.RequestStart(t.Context(), pending.RunID, time.Now(), time.Minute, events[waiting], limit.MustNew(1))
	require.NoError(t, err)
	require.Equal(t, core.Changed(true), changed)
}
