package user_sessions

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

func (r *Repository) Login(ctx context.Context, session domain.UserSession) error {
	log := logger.FromContext(ctx)
	raw := FromModel(session)

	q := r.sb.Insert(UserSessionsTable).
		Values(raw.Values()...).
		Suffix(`
			ON CONFLICT (user_id, fingerprint)
			DO UPDATE SET
				token_sha256 = EXCLUDED.token_sha256,
				created_at = EXCLUDED.created_at,
				expires_at = EXCLUDED.expires_at,
				id = EXCLUDED.id
		`)

	tag, err := r.pool.Execx(ctx, q)
	if err != nil {
		return pgerr.ScanErr(err)
	}
	if tag.RowsAffected() != 1 {
		log.Info().Msg("postgres: conflict on insert session")
	}
	return nil
}
