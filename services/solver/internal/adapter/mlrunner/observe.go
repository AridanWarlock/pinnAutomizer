package mlrunner

import (
	"context"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

// ObserveAttempt returns a copy; it neither starts containers nor writes the journal.
func (r *PinnRunner) ObserveAttempt(ctx context.Context, a domain.Attempt) (domain.Attempt, error) {
	if !a.CanObserve() {
		return domain.Attempt{}, errs.ErrConflict
	}
	c, err := r.inspectAttempt(ctx, a.ContainerID, a)
	if err != nil {
		return domain.Attempt{}, err
	}
	if c.Running {
		return a, nil
	}
	if !c.IsExited() || !c.HasStarted() {
		return domain.Attempt{}, fmt.Errorf("%w: cannot determine execution result in container state %q", domain.ErrStartOutcomeUnknown, c.State)
	}
	var failure *string
	if c.ExitCode != 0 || c.Error != "" {
		message := fmt.Sprintf("container exited with code %d", c.ExitCode)
		if c.Error != "" {
			message += ": " + c.Error
		}
		failure = &message
	}
	if _, err := a.Finish(&c.ExitCode, failure, time.Now()); err != nil {
		return domain.Attempt{}, err
	}
	return a, nil
}
