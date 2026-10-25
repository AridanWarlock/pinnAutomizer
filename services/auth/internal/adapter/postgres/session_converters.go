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

type SessionRevocationRaw struct {
	Jti         core.Jti         `db:"jti"`
	UserID      uuid.UUID        `db:"user_id"`
	Fingerprint core.Fingerprint `db:"fingerprint"`
}

func newSessionRevocationRaw(
	userID uuid.UUID,
	fingerprint core.Fingerprint,
	jti core.Jti,
) SessionRevocationRaw {
	return SessionRevocationRaw{
		Jti:         jti,
		UserID:      userID,
		Fingerprint: fingerprint,
	}
}

func (r SessionRevocationRaw) Values() []any {
	return []any{
		r.Jti,
		r.UserID,
		r.Fingerprint,
	}
}
