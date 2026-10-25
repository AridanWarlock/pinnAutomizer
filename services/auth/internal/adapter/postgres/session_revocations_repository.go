package postgres

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
)

func (r *Repository) QueueSessionRevocation(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint core.Fingerprint,
	jti core.Jti,
) error {
	q := r.sb.Insert(SessionRevocationsTable).
		Columns(SessionRevocationsJti, SessionRevocationsUserID, SessionRevocationsFingerprint).
		Values(uuid.UUID(jti), userID, fingerprint).
		Suffix("ON CONFLICT (jti) DO NOTHING")
	_, err := r.pool.Execx(ctx, q)
	return err
}

func (r *Repository) GetSessionRevocations(
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

func (r *Repository) GetPendingSessionRevocations(
	ctx context.Context,
	after uuid.UUID,
	batchSize limit.Limit,
) ([]core.Jti, error) {
	q := r.sb.Select(SessionRevocationsJti).
		From(SessionRevocationsTable).
		Where(sq.Gt{SessionRevocationsJti: after}).
		OrderBy(SessionRevocationsJti).
		Limit(batchSize.Uint64())
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
	q := r.sb.Delete(SessionRevocationsTable).
		Where(sq.Eq{SessionRevocationsJti: uuid.UUID(jti)})
	_, err := r.pool.Execx(ctx, q)
	return err
}

func (r *Repository) checkSessionRevocationsSchema(ctx context.Context) error {
	q := r.sb.Select(
		SessionRevocationsJti,
		SessionRevocationsUserID,
		SessionRevocationsFingerprint,
	).
		From(SessionRevocationsTable).
		Limit(0)
	_, err := r.pool.Execx(ctx, q)
	return err
}
