package corefixtures

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func NewJti() core.Jti {
	return core.Jti(uuid.New())
}
