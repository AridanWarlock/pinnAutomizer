package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestInvalidRefreshTokenRowFailsFast(t *testing.T) {
	require.Panics(t, func() { _ = toRefreshTokenModel(RefreshTokenRaw{}) })
}

func TestInvalidSessionJtiPanics(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() { toJtiModel(uuid.Nil) })
}
