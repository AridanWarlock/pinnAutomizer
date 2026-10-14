package corefixtures

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func NewIdempotencyKey() core.IdempotencyKey {
	return core.IdempotencyKey(uuid.NewString())
}
