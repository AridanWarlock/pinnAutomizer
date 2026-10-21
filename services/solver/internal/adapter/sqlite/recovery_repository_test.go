package sqlite

import (
	"cmp"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain/fixtures"
)

func TestUnfinishedPaginationDoesNotSkipAfterEarlierRowsFinish(t *testing.T) {
	cfg := config(t)
	repo := open(t, cfg)
	attempts := []domain.Attempt{fixtures.NewAttempt(t), fixtures.NewAttempt(t), fixtures.NewAttempt(t), fixtures.NewAttempt(t)}
	slices.SortFunc(attempts, func(a, b domain.Attempt) int { return cmp.Compare(a.RunID.String(), b.RunID.String()) })
	for _, a := range attempts {
		_, _, err := repo.RegisterAttempt(t.Context(), a)
		require.NoError(t, err)
	}
	first, err := repo.GetRecoverableAttempts(t.Context(), uuid.Nil, limit.MustNew(2))
	require.NoError(t, err)
	require.Equal(t, attempts[:2], first)
	for _, a := range first {
		code := 0
		_, err := repo.FinishAttempt(t.Context(), a.RunID, &code, nil, time.Now(), event(t, a))
		require.NoError(t, err)
	}
	require.NoError(t, repo.Close())
	repo = open(t, cfg)
	remaining, err := repo.GetRecoverableAttempts(t.Context(), uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Equal(t, attempts[2:], remaining)
	next, err := repo.GetRecoverableAttempts(t.Context(), first[1].RunID, limit.MustNew(2))
	require.NoError(t, err)
	require.Equal(t, attempts[2:], next)
	empty, err := repo.GetRecoverableAttempts(t.Context(), next[1].RunID, limit.MustNew(2))
	require.NoError(t, err)
	require.Empty(t, empty)
}
