package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	shared "github.com/AridanWarlock/pinnAutomizer/pkg/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type Repository struct {
	sqlite *shared.SQLite
	sb     sq.StatementBuilderType
}

var ErrAlreadyOwned = shared.ErrAlreadyOwned

func New(cfg Config) (_ *Repository, err error) {
	client, err := shared.New(shared.Config{Path: cfg.Path, BusyTimeout: cfg.BusyTimeout})
	if err != nil {
		return nil, err
	}
	ready := false
	defer func() {
		if !ready {
			err = errors.Join(err, client.Close())
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = client.CheckMigrations(ctx, os.DirFS(cfg.MigrationsDir)); err != nil {
		return nil, err
	}
	ready = true
	return &Repository{sqlite: client, sb: sq.StatementBuilder.PlaceholderFormat(sq.Question)}, nil
}

func (r *Repository) Close() error {
	return r.sqlite.Close()
}

func (r *Repository) GetAttempt(ctx context.Context, id uuid.UUID) (domain.Attempt, error) {
	query := r.sb.Select(attemptColumns...).From(attemptsTable).Where(sq.Eq{"run_id": id.String()})
	var row attemptRow
	if err := r.sqlite.Getx(ctx, &row, query); err != nil {
		return domain.Attempt{}, err
	}
	return row.toDomain(), nil
}

func (r *Repository) RegisterAttempt(ctx context.Context, a domain.Attempt) (domain.Attempt, bool, error) {
	valid, err := domain.NewAttempt(a.RunID, a.TaskID, a.Mode, a.CreatedAt)
	if err != nil {
		return domain.Attempt{}, false, err
	}
	row := fromAttempt(valid)
	query := r.sb.Insert(attemptsTable).Columns(attemptColumns...).Values(row.Values()...).Suffix("ON CONFLICT(run_id) DO NOTHING")
	tag, err := r.sqlite.Execx(ctx, query)
	if err != nil {
		return domain.Attempt{}, false, err
	}
	stored, err := r.GetAttempt(ctx, a.RunID)
	if err == nil && !stored.Matches(a) {
		err = errs.ErrConflict
	}
	return stored, tag.RowsAffected() == 1, err
}

func (r *Repository) FinishAttempt(ctx context.Context, id uuid.UUID, code *int, failure *string, now time.Time, event domain.Event) (bool, error) {
	if now.IsZero() || (failure == nil && (code == nil || *code != 0)) || event.RunID != id {
		return false, errs.ErrInvalidArgument
	}
	if err := event.Validate(); err != nil {
		return false, err
	}
	headers, err := json.Marshal(event.Message.Headers)
	if err != nil {
		return false, err
	}
	changed := false
	err = r.sqlite.InTransaction(ctx, func(ctx context.Context) error {
		update := r.sb.Update(attemptsTable).SetMap(sq.Eq{"state": domain.AttemptFinished, "completed_at": now.UTC().UnixNano(), "exit_code": code, "error": failure}).Where(sq.Eq{"run_id": id.String()}).Where(sq.NotEq{"state": domain.AttemptFinished})
		tag, err := r.sqlite.Execx(ctx, update)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			stored, err := r.GetAttempt(ctx, id)
			if err != nil {
				return err
			}
			if stored.State != domain.AttemptFinished {
				return errs.ErrConflict
			}
			return nil
		}
		insert := r.sb.Insert(outboxTable).Columns(eventColumns...).Values(event.ID.String(), id.String(), event.Message.Topic, event.Message.Key, event.Message.Value, string(headers))
		if _, err = r.sqlite.Execx(ctx, insert); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

func (r *Repository) GetPendingEvents(ctx context.Context, limit int) ([]domain.Event, error) {
	if limit <= 0 {
		return nil, errs.ErrInvalidArgument
	}
	query := r.sb.Select(eventColumns...).From(outboxTable).OrderBy("sequence").Limit(uint64(limit))
	var rows []eventRow
	if err := r.sqlite.Selectx(ctx, &rows, query); err != nil {
		return nil, err
	}
	events := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		var headers core.KafkaHeaders
		if err := json.Unmarshal([]byte(row.Headers), &headers); err != nil {
			return nil, err
		}
		events = append(events, domain.Event{ID: row.ID, RunID: row.RunID, Message: core.NewProduceKafkaMessage(row.Topic, row.Key, row.Value, headers)})
	}
	return events, nil
}

func (r *Repository) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	_, err := r.sqlite.Execx(ctx, r.sb.Delete(outboxTable).Where(sq.Eq{"event_id": id.String()}))
	return err
}

func (r *Repository) RecordContainer(ctx context.Context, id uuid.UUID, container string) (bool, error) {
	if container == "" {
		return false, errs.ErrInvalidArgument
	}
	query := r.sb.Update(attemptsTable).SetMap(sq.Eq{"state": domain.AttemptContainerCreated, "container_id": container}).Where(sq.Eq{"run_id": id.String(), "state": domain.AttemptAccepted})
	tag, err := r.sqlite.Execx(ctx, query)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	stored, err := r.GetAttempt(ctx, id)
	if err != nil {
		return false, err
	}
	if stored.ContainerID != container {
		return false, errs.ErrConflict
	}
	return false, nil
}

// RequestStart records intent before Docker Start. A replay must never start again.
func (r *Repository) RequestStart(ctx context.Context, id uuid.UUID, now time.Time, timeout time.Duration) (bool, error) {
	if now.IsZero() || timeout <= 0 {
		return false, errs.ErrInvalidArgument
	}
	query := r.sb.Update(attemptsTable).SetMap(sq.Eq{"state": domain.AttemptStartRequested, "deadline": now.Add(timeout).UTC().UnixNano()}).Where(sq.Eq{"run_id": id.String(), "state": domain.AttemptContainerCreated})
	tag, err := r.sqlite.Execx(ctx, query)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	stored, err := r.GetAttempt(ctx, id)
	if err != nil {
		return false, err
	}
	if stored.State != domain.AttemptStartRequested && stored.State != domain.AttemptRunning && stored.State != domain.AttemptFinished {
		return false, errs.ErrConflict
	}
	return false, nil
}
