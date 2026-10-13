package domain

import (
	"strings"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskRunRequestValidatesKey(t *testing.T) {
	for _, size := range []int{0, TaskRunRequestIdKeyMaxBytes, TaskRunRequestIdKeyMaxBytes + 1} {
		request, err := NewTaskRunRequest(uuid.New(), uuid.New(), core.IdempotencyKey(strings.Repeat("k", size)))
		if size == TaskRunRequestIdKeyMaxBytes {
			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, request.EventID)
		} else {
			require.ErrorIs(t, err, errs.ErrInvalidArgument)
		}
	}
}
