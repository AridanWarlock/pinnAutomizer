package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/google/uuid"
)

func NewJti() domain.Jti {
	return domain.Jti(uuid.New())
}
