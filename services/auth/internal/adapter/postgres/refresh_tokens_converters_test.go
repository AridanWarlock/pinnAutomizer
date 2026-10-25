package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInvalidRefreshTokenRowFailsFast(t *testing.T) {
	require.Panics(t, func() { _ = toRefreshTokenModel(RefreshTokenRaw{}) })
}
