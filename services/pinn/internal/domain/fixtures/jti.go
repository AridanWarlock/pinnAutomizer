package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain"
	"github.com/google/uuid"
)

func NewJti() domain.Jti {
	return domain.Jti(uuid.New())
}
