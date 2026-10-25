package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/goRedis"
)

func TestRedisFailureClassification(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		input       error
		cause       error
		unavailable bool
	}{
		{
			name:        "missing key",
			input:       goRedis.ErrKeyNotFound,
			cause:       errs.ErrKeyNotFound,
			unavailable: false,
		},
		{
			name:        "closed client",
			input:       goRedis.ErrClosed,
			cause:       errs.ErrClosed,
			unavailable: true,
		},
		{
			name:        "pool exhausted",
			input:       goRedis.ErrPoolExhausted,
			cause:       errs.ErrPoolExhausted,
			unavailable: true,
		},
		{
			name:        "backend failure",
			input:       fmt.Errorf("connection reset: %w", errs.ErrServiceUnavailable),
			cause:       errs.ErrServiceUnavailable,
			unavailable: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := (&Redis{}).handleCommonErr(tt.input)
			require.ErrorIs(t, err, tt.cause)
			require.Equal(t, tt.unavailable, errors.Is(err, errs.ErrServiceUnavailable), "input=%v", tt.input)
		})
	}
}

func TestSerializationFailureIsNotServiceUnavailable(t *testing.T) {
	t.Parallel()
	for _, input := range []error{
		&json.UnsupportedTypeError{Type: reflect.TypeFor[chan int]()},
		&json.SyntaxError{Offset: 1},
	} {
		err := (&Redis{}).handleCommonErr(input)
		require.ErrorIs(t, err, input)
		require.NotErrorIs(t, err, errs.ErrServiceUnavailable)
	}
}
