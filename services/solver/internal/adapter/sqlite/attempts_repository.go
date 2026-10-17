package sqlite

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func (r *Repository) GetAttempt(ctx context.Context, id uuid.UUID) (domain.Attempt, error) {
	query := r.sb.Select(attemptColumns...).
		From(attemptsTable).
		Where(sq.Eq{attemptsRunID: id})
	var row attemptRow
	if err := r.sqlite.Getx(ctx, &row, query); err != nil {
		return domain.Attempt{}, err
	}
	return row.toDomain()
}

func (r *Repository) RegisterAttempt(ctx context.Context, a domain.Attempt) (stored domain.Attempt, created domain.Created, err error) {
	if err := a.Validate(); err != nil {
		return domain.Attempt{}, false, err
	}
	if a.State != domain.AttemptAccepted {
		return domain.Attempt{}, false, errs.ErrInvalidArgument
	}
	err = r.sqlite.InTransaction(ctx, func(ctx context.Context) error {
		query := r.sb.Insert(attemptsTable).
			Columns(attemptColumns...).
			Values(fromAttempt(a).Values()...).
			Suffix("ON CONFLICT(" + attemptsRunID + ") DO NOTHING").
			Suffix("RETURNING " + joinColumns(attemptColumns))
		var row attemptRow
		insertErr := r.sqlite.Getx(ctx, &row, query)
		if insertErr == nil {
			var err error
			stored, err = row.toDomain()
			created = true
			return err
		}
		if !errors.Is(insertErr, errs.ErrNotFound) {
			return insertErr
		}
		// A replay needs the stored command, not just a generic unique-key error.
		var err error
		stored, err = r.GetAttempt(ctx, a.RunID)
		if err != nil {
			return err
		}
		if !stored.Matches(a) {
			return errs.ErrConflict
		}
		return nil
	})
	if err != nil {
		return domain.Attempt{}, false, err
	}
	return stored, created, nil
}

func (r *Repository) RecordContainer(ctx context.Context, id uuid.UUID, containerID string) (changed domain.Changed, err error) {
	return r.transition(ctx, id, func(a *domain.Attempt) (domain.Changed, error) {
		return a.RecordContainer(containerID)
	})
}

func (r *Repository) RequestStart(ctx context.Context, id uuid.UUID, now time.Time, timeout time.Duration) (changed domain.Changed, err error) {
	return r.transition(ctx, id, func(a *domain.Attempt) (domain.Changed, error) {
		return a.RequestStart(now, timeout)
	})
}

func (r *Repository) transition(ctx context.Context, id uuid.UUID, change func(*domain.Attempt) (domain.Changed, error)) (domain.Changed, error) {
	var changed domain.Changed
	err := r.sqlite.InTransaction(ctx, func(ctx context.Context) error {
		a, err := r.GetAttempt(ctx, id)
		if err != nil {
			return err
		}
		previous := a.State
		changed, err = change(&a)
		if err != nil || !changed {
			return err
		}
		return r.saveTransition(ctx, a, previous)
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

func (r *Repository) saveTransition(ctx context.Context, a domain.Attempt, previous domain.AttemptState) error {
	row := fromAttempt(a)
	query := r.sb.Update(attemptsTable).
		SetMap(sq.Eq{
			attemptsState:       row.State,
			attemptsContainerID: row.ContainerID,
			attemptsDeadline:    row.Deadline,
			attemptsCompletedAt: row.CompletedAt,
			attemptsExitCode:    row.ExitCode,
			attemptsError:       row.Error,
		}).
		Where(sq.Eq{attemptsRunID: row.RunID, attemptsState: previous})
	tag, err := r.sqlite.Execx(ctx, query)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errs.ErrConflict
	}
	return nil
}
