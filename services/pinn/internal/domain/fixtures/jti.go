package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/google/uuid"
)

func NewJti() core.Jti {
	return core.Jti(uuid.New())
}
