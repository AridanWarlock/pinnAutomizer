package mlrunner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

const (
	stopTimeout     = 10 * time.Second
	stopGracePeriod = 5 * time.Second
)

// ObserveAttempt returns a copy. It may stop an overdue running container,
// but never starts containers or writes the journal.
func (r *PinnRunner) ObserveAttempt(ctx context.Context, a domain.Attempt) (domain.Attempt, error) {
	if !a.CanObserve() {
		return domain.Attempt{}, errs.ErrConflict
	}
	c, err := r.inspectAttempt(ctx, a.ContainerID, a)
	if err != nil {
		return domain.Attempt{}, err
	}
	timedOut := false
	if c.Running {
		if !a.DeadlineExceeded(time.Now()) {
			return a, nil
		}
		c, err = r.stopExpired(a)
		if err != nil {
			return domain.Attempt{}, err
		}
		timedOut = true
	}
	if !c.IsExited() || !c.HasStarted() || c.FinishedAt.IsZero() {
		return domain.Attempt{}, fmt.Errorf("%w: cannot determine execution result in container state %q", domain.ErrStartOutcomeUnknown, c.State)
	}
	if timedOut || a.DeadlineExceeded(c.FinishedAt) {
		_, err := a.FinishTimedOut(c.ExitCode, time.Now())
		return a, err
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

func (r *PinnRunner) stopExpired(a domain.Attempt) (docker.Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	stopErr := r.docker.Stop(ctx, a.ContainerID, stopGracePeriod)
	c, inspectErr := r.inspectAttempt(ctx, a.ContainerID, a)
	if inspectErr != nil || !c.IsExited() || !c.HasStarted() || c.FinishedAt.IsZero() {
		return docker.Container{}, fmt.Errorf("%w: %w", domain.ErrStopOutcomeUnknown,
			errors.Join(stopErr, inspectErr, fmt.Errorf("exit not confirmed in state %q", c.State)))
	}
	return c, nil
}
