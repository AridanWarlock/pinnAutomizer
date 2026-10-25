package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func (r *Repository) QueueSessionRevocation(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
	jti core.Jti,
) error {
	raw := newSessionRevocationRaw(userID, fingerprint, jti)
	q := r.sb.Insert(SessionRevocationsTable).
		Columns(SessionRevocationsColumns...).
		Values(raw.Values()...).
		Suffix("ON CONFLICT (jti) DO NOTHING")
	_, err := r.pool.Execx(ctx, q)
	return err
}

func (r *Repository) GetSessionRevocationJtis(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
) ([]core.Jti, error) {
	q := r.sb.Select(SessionRevocationsJti).
		From(SessionRevocationsTable).
		Where(sq.Eq{
			SessionRevocationsUserID:      userID,
			SessionRevocationsFingerprint: fingerprint,
		}).
		OrderBy(SessionRevocationsJti)
	return r.getRevocationJtis(ctx, q)
}

func (r *Repository) GetPendingSessionRevocationJtis(
	ctx context.Context,
	after *core.Jti,
	batchSize limit.Limit,
) ([]core.Jti, error) {
	q := r.sb.Select(SessionRevocationsJti).
		From(SessionRevocationsTable).
		OrderBy(SessionRevocationsJti).
		Limit(batchSize.Uint64())
	if after != nil {
		q = q.Where(sq.Gt{SessionRevocationsJti: uuid.UUID(*after)})
	}
	return r.getRevocationJtis(ctx, q)
}

func (r *Repository) getRevocationJtis(ctx context.Context, q sq.SelectBuilder) ([]core.Jti, error) {
	var raw []uuid.UUID
	if err := r.pool.Selectx(ctx, &raw, q); err != nil {
		return nil, err
	}
	return toJtiModels(raw), nil
}

func (r *Repository) ConfirmSessionRevocation(ctx context.Context, jti core.Jti) error {
	// UUID implements Valuer; Squirrel otherwise expands core.Jti's array into an IN list.
	q := r.sb.Delete(SessionRevocationsTable).
		Where(sq.Eq{SessionRevocationsJti: uuid.UUID(jti)})
	_, err := r.pool.Execx(ctx, q)
	return err
}

func (r *Repository) RevokeSessions(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
	requestJti core.Jti,
) error {
	return r.InTransaction(ctx, func(ctx context.Context) error {
		return r.revokeSessions(ctx, userID, fingerprint, requestJti)
	})
}

func (r *Repository) revokeSessions(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
	requestJti core.Jti,
) error {
	currentJti, err := r.GetJtiByFingerprintForUpdate(ctx, userID, fingerprint)
	if errors.Is(err, errs.ErrNotFound) {
		return r.QueueSessionRevocation(ctx, userID, fingerprint, requestJti)
	}
	if err != nil {
		return fmt.Errorf("lock current session: %w", err)
	}
	if currentJti != requestJti {
		if err := r.QueueSessionRevocation(ctx, userID, fingerprint, currentJti); err != nil {
			return err
		}
	}
	if err := r.QueueSessionRevocation(ctx, userID, fingerprint, requestJti); err != nil {
		return err
	}
	return r.Logout(ctx, userID, fingerprint)
}
