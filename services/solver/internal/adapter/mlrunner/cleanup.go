package mlrunner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

const cleanupTimeout = 10 * time.Second

// CleanupAttempt only accepts a durable finished attempt; it never stops or starts work.
func (r *PinnRunner) CleanupAttempt(ctx context.Context, a domain.Attempt) error {
	if !a.IsFinished() {
		return errs.ErrConflict
	}
	if !a.NeedsCleanup() {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()
	c, err := r.inspectAttempt(ctx, a.ContainerID, a)
	if errors.Is(err, errs.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !c.IsFinished() {
		return fmt.Errorf("%w: cannot clean container in state %q", errs.ErrConflict, c.State)
	}
	err = r.docker.RemoveStopped(ctx, a.ContainerID)
	if errors.Is(err, errs.ErrNotFound) {
		return nil
	}
	return err
}
