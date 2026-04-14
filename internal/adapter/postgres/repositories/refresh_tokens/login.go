package refresh_tokens

import (
	"context"
	"strings"

	. "github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres/schema"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

func (r *Repository) Login(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error) {
	raw := FromModel(token)

	q := r.sb.Insert(RefreshTokensTable).
		Values(raw.Values()...).
		Suffix(`
			ON CONFLICT (user_id, fingerprint)
			DO UPDATE SET
				hash = EXCLUDED.hash,
				jti = EXCLUDED.jti,
				agent = EXCLUDED.agent,
				ip = EXCLUDED.ip,
				created_at = EXCLUDED.created_at,
				expires_at = EXCLUDED.expires_at
			RETURNING ` + strings.Join(RefreshTokensTableColumns, ","))

	var outRow RefreshTokenRaw
	if err := r.pool.Getx(ctx, &outRow, q); err != nil {
		return domain.RefreshToken{}, err
	}
	return ToModel(outRow), nil
}
