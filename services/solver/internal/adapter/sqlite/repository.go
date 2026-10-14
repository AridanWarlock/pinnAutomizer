package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"
	driver "modernc.org/sqlite"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	shared "github.com/AridanWarlock/pinnAutomizer/pkg/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type Repository struct {
	db     *sql.DB
	client *shared.SQLite
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
	return &Repository{db: client.Database(), client: client}, nil
}
func (r *Repository) Close() error { return r.client.Close() }

const columns = "run_id,task_id,mode,state,container_id,created_at,deadline,completed_at,exit_code,error"

type scanner interface{ Scan(...any) error }

func scanAttempt(row scanner) (domain.Attempt, error) {
	var a domain.Attempt
	var run, task string
	var created int64
	var deadline, completed, code sql.NullInt64
	var failure sql.NullString
	err := row.Scan(&run, &task, &a.Mode, &a.State, &a.ContainerID, &created, &deadline, &completed, &code, &failure)
	if errors.Is(err, sql.ErrNoRows) {
		return a, errs.ErrNotFound
	}
	if err != nil {
		return a, err
	}
	if a.RunID, err = uuid.Parse(run); err != nil {
		return a, err
	}
	if a.TaskID, err = uuid.Parse(task); err != nil {
		return a, err
	}
	a.CreatedAt = time.Unix(0, created).UTC()
	if deadline.Valid {
		v := time.Unix(0, deadline.Int64).UTC()
		a.Deadline = &v
	}
	if completed.Valid {
		v := time.Unix(0, completed.Int64).UTC()
		a.CompletedAt = &v
	}
	if code.Valid {
		v := int(code.Int64)
		a.ExitCode = &v
	}
	if failure.Valid {
		a.Error = &failure.String
	}
	return a, nil
}

func (r *Repository) GetAttempt(ctx context.Context, id uuid.UUID) (domain.Attempt, error) {
	return scanAttempt(r.db.QueryRowContext(ctx, "SELECT "+columns+" FROM attempts WHERE run_id=?", id.String()))
}

func (r *Repository) RegisterAttempt(ctx context.Context, a domain.Attempt) (domain.Attempt, bool, error) {
	valid, err := domain.NewAttempt(a.RunID, a.TaskID, a.Mode, a.CreatedAt)
	if err != nil {
		return domain.Attempt{}, false, err
	}
	result, err := r.db.ExecContext(ctx, "INSERT INTO attempts(run_id,task_id,mode,state,created_at) VALUES(?,?,?,?,?) ON CONFLICT(run_id) DO NOTHING", valid.RunID.String(), valid.TaskID.String(), valid.Mode, valid.State, valid.CreatedAt.UnixNano())
	if err != nil {
		var e *driver.Error
		if errors.As(err, &e) && (e.Code() == 1555 || e.Code() == 2067) {
			err = errs.ErrConflict
		}
		return domain.Attempt{}, false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return domain.Attempt{}, false, err
	}
	stored, err := r.GetAttempt(ctx, a.RunID)
	if err == nil && !stored.Matches(a) {
		err = errs.ErrConflict
	}
	return stored, n == 1, err
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, "UPDATE attempts SET state='finished',completed_at=?,exit_code=?,error=? WHERE run_id=? AND state!='finished'", now.UTC().UnixNano(), code, failure, id.String())
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		a, e := scanAttempt(tx.QueryRowContext(ctx, "SELECT "+columns+" FROM attempts WHERE run_id=?", id.String()))
		if e != nil {
			return false, e
		}
		if a.State != domain.AttemptFinished {
			return false, errs.ErrConflict
		}
		return false, nil
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO outbox(event_id,run_id,topic,message_key,value,headers) VALUES(?,?,?,?,?,?)", event.ID.String(), id.String(), event.Message.Topic, event.Message.Key, event.Message.Value, string(headers))
	if err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (r *Repository) GetPendingEvents(ctx context.Context, limit int) ([]domain.Event, error) {
	if limit <= 0 {
		return nil, errs.ErrInvalidArgument
	}
	rows, err := r.db.QueryContext(ctx, "SELECT event_id,run_id,topic,message_key,value,headers FROM outbox ORDER BY sequence LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.Event
	for rows.Next() {
		var e domain.Event
		var id, run, headers string
		if err = rows.Scan(&id, &run, &e.Message.Topic, &e.Message.Key, &e.Message.Value, &headers); err != nil {
			return nil, err
		}
		if e.ID, err = uuid.Parse(id); err != nil {
			return nil, err
		}
		if e.RunID, err = uuid.Parse(run); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(headers), &e.Message.Headers); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *Repository) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM outbox WHERE event_id=?", id.String())
	return err
}

func (r *Repository) RecordContainer(ctx context.Context, id uuid.UUID, container string) (bool, error) {
	if container == "" {
		return false, errs.ErrInvalidArgument
	}
	result, err := r.db.ExecContext(ctx, "UPDATE attempts SET state='created',container_id=? WHERE run_id=? AND state='registered'", container, id.String())
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n == 1 {
		return n == 1, err
	}
	a, err := r.GetAttempt(ctx, id)
	if err != nil {
		return false, err
	}
	if a.ContainerID != container {
		return false, errs.ErrConflict
	}
	return false, nil
}

// RequestStart records intent before Docker Start. A replay must never start again.
func (r *Repository) RequestStart(ctx context.Context, id uuid.UUID, now time.Time, timeout time.Duration) (bool, error) {
	if now.IsZero() || timeout <= 0 {
		return false, errs.ErrInvalidArgument
	}
	result, err := r.db.ExecContext(ctx, "UPDATE attempts SET state='start_requested',deadline=? WHERE run_id=? AND state='created'", now.Add(timeout).UTC().UnixNano(), id.String())
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil || n == 1 {
		return n == 1, err
	}
	a, err := r.GetAttempt(ctx, id)
	if err != nil {
		return false, err
	}
	if a.State != domain.AttemptStartRequested && a.State != domain.AttemptRunning && a.State != domain.AttemptFinished {
		return false, errs.ErrConflict
	}
	return false, nil
}
