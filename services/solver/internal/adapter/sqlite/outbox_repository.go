package sqlite

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

// FinishAttempt stores the result and its outbox event in one transaction.
func (r *Repository) FinishAttempt(ctx context.Context, id uuid.UUID, code *int, failure *string, now time.Time, event domain.Event) (domain.Changed, error) {
	if event.RunID != id {
		return false, errs.ErrInvalidArgument
	}
	if err := event.Validate(); err != nil {
		return false, err
	}
	row, err := fromEvent(event)
	if err != nil {
		return false, err
	}
	var changed domain.Changed
	err = r.sqlite.InTransaction(ctx, func(ctx context.Context) error {
		a, err := r.GetAttempt(ctx, id)
		if err != nil {
			return err
		}
		previous := a.State
		changed, err = a.Finish(code, failure, now)
		if err != nil || !changed {
			return err
		}
		if err := r.saveTransition(ctx, a, previous); err != nil {
			return err
		}
		query := r.sb.Insert(outboxTable).
			Columns(eventColumns...).
			Values(row.Values()...)
		_, err = r.sqlite.Execx(ctx, query)
		return err
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

func (r *Repository) GetPendingEvents(ctx context.Context, batch limit.Limit) ([]domain.Event, error) {
	if err := batch.Validate(); err != nil {
		return nil, err
	}
	query := r.sb.Select(eventColumns...).
		From(outboxTable).
		OrderBy(outboxSequence).
		Limit(batch.Uint64())
	var rows []eventRow
	if err := r.sqlite.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}
	events := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		event, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *Repository) DeleteEvents(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	query := r.sb.Delete(outboxTable).
		Where(sq.Eq{outboxID: ids})
	_, err := r.sqlite.Execx(ctx, query)
	return err
}
