package limit

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestLimitRequiresPositiveValue(t *testing.T) {
	for _, value := range []int{-1, 0, 1, 32} {
		t.Run(strconv.Itoa(value), func(t *testing.T) {
			l, err := New(value)
			if value <= 0 {
				require.ErrorIs(t, err, errs.ErrInvalidArgument)
				return
			}
			require.NoError(t, err)
			require.Equal(t, value, l.Int())
			require.Equal(t, uint64(value), l.Uint64())
		})
	}
	require.ErrorIs(t, (Limit{}).Validate(), errs.ErrInvalidArgument)
	require.Panics(t, func() {
		MustNew(0)
	})
}
