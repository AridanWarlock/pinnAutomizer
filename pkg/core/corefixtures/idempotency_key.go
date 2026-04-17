package corefixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewIdempotencyKey() core.IdempotencyKey {
	return core.IdempotencyKey(uuid.NewString())
}
