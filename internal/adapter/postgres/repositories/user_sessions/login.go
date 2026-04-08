package user_sessions

import (
	"context"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/pgerr"
	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

func (r *Repository) Login(ctx context.Context, session domain.UserSession) (domain.UserSession, error) {
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
			RETURNING ` + strings.Join(UserSessionsTableColumns, ","))

	var outRow UserSessionRaw
	if err := r.pool.Getx(ctx, &outRow, q); err != nil {
		return domain.UserSession{}, pgerr.ScanErr(err)
	}
	return ToModel(outRow), nil
}
