package postgres

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func toJtiModel(raw uuid.UUID) core.Jti {
	jti, err := core.NewJti(raw)
	if err != nil {
		panic(fmt.Errorf("postgres: invalid session JTI: %w", err))
	}
	return jti
}

func toJtiModels(raw []uuid.UUID) []core.Jti {
	jtis := make([]core.Jti, len(raw))
	for i, id := range raw {
		jtis[i] = toJtiModel(id)
	}
	return jtis
}
