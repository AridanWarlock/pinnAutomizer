package postgres

import (
	"context"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	q := r.sb.Select(RefreshTokensColumns...).
		From(RefreshTokensTable).
		Where(sq.Eq{RefreshTokensHash: hash})

	var out RefreshTokenRaw
	if err := r.pool.Getx(ctx, &out, q); err != nil {
		return domain.RefreshToken{}, err
	}

	return toRefreshTokenModel(out), nil
}

func (r *Repository) GetJtiByFingerprint(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
) (core.Jti, error) {
	q := r.sb.Select(RefreshTokensJti).
		From(RefreshTokensTable).
		Where(sq.Eq{
			RefreshTokensUserID:      userID,
			RefreshTokensFingerprint: fingerprint,
		})

	var jti core.Jti
	if err := r.pool.Getx(ctx, &jti, q); err != nil {
		return core.Jti{}, err
	}
	return jti, nil
}

func (r *Repository) Login(ctx context.Context, token domain.RefreshToken) (domain.RefreshToken, error) {
	raw := fromRefreshTokenModel(token)

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
			RETURNING ` + strings.Join(RefreshTokensColumns, ","))

	var outRow RefreshTokenRaw
	if err := r.pool.Getx(ctx, &outRow, q); err != nil {
		return domain.RefreshToken{}, err
	}
	return toRefreshTokenModel(outRow), nil
}

func (r *Repository) Logout(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
) error {
	q := r.sb.Delete(RefreshTokensTable).
		Where(sq.Eq{
			RefreshTokensUserID:      userID,
			RefreshTokensFingerprint: fingerprint,
		})

	tag, err := r.pool.Execx(ctx, q)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *Repository) RotateRefreshToken(
	ctx context.Context,
	oldHash string,
	newHash string,
	newJti core.Jti,
) error {
	q := r.sb.Update(RefreshTokensTable).
		Set(RefreshTokensHash, newHash).
		Set(RefreshTokensJti, newJti).
		Where(sq.Eq{RefreshTokensHash: oldHash})

	tag, err := r.pool.Execx(ctx, q)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errs.ErrNotFound
	}
	return nil
}
