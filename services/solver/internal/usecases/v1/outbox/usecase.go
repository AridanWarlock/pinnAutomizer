package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type Repository interface {
	GetPendingEvents(context.Context, int) ([]domain.Event, error)
	DeleteEvent(context.Context, uuid.UUID) error
}

// Writer must confirm delivery synchronously. Async writers cannot drain an outbox safely.
type Writer interface {
	WriteMessages(context.Context, ...core.KafkaMessage) error
}
type usecase struct {
	repo   Repository
	writer Writer
}

func New(repo Repository, writer Writer) *usecase { return &usecase{repo: repo, writer: writer} }

func (u *usecase) PublishPending(ctx context.Context) error {
	events, err := u.repo.GetPendingEvents(ctx, 32)
	if err != nil {
		return fmt.Errorf("read solver outbox: %w", err)
	}
	for _, e := range events {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = u.writer.WriteMessages(ctx, e.Message); err != nil {
			return fmt.Errorf("publish solver event %s: %w", e.ID, err)
		}
		if err = u.repo.DeleteEvent(ctx, e.ID); err != nil {
			return fmt.Errorf("acknowledge solver event %s: %w", e.ID, err)
		}
	}
	return nil
}

// Run has no detached goroutines: the caller owns it and waits for shutdown.
// Transient errors remain in the journal and are reported once per retry.
func (u *usecase) Run(ctx context.Context, interval time.Duration, report func(error)) error {
	if interval <= 0 || report == nil {
		return errs.ErrInvalidArgument
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := u.PublishPending(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			report(err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
