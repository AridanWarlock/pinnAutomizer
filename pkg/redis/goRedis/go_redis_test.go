package goRedis

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestCommandFailureClassification(t *testing.T) {
	t.Parallel()
	networkFailure := errors.New("connection reset")
	for _, tt := range []struct {
		name        string
		input       error
		cause       error
		unavailable bool
	}{
		{
			name:        "missing key",
			input:       redis.Nil,
			cause:       ErrKeyNotFound,
			unavailable: false,
		},
		{
			name:        "closed client",
			input:       redis.ErrClosed,
			cause:       ErrClosed,
			unavailable: false,
		},
		{
			name:        "pool exhausted",
			input:       redis.ErrPoolTimeout,
			cause:       ErrPoolExhausted,
			unavailable: false,
		},
		{
			name:        "cancelled",
			input:       context.Canceled,
			cause:       context.Canceled,
			unavailable: true,
		},
		{
			name:        "deadline",
			input:       context.DeadlineExceeded,
			cause:       context.DeadlineExceeded,
			unavailable: true,
		},
		{
			name:        "network failure",
			input:       networkFailure,
			cause:       networkFailure,
			unavailable: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := (&Client{}).handleCommonErr(fmt.Errorf("command: %w", tt.input))
			require.ErrorIs(t, err, tt.cause)
			require.Equal(t, tt.unavailable, errors.Is(err, errs.ErrServiceUnavailable), "input=%v", tt.input)
		})
	}
}
