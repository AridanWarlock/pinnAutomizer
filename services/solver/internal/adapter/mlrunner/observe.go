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
	if c.IsRunning() {
		if !a.DeadlineExceeded(time.Now()) {
			return a, nil
		}
		c, err = r.stopExpired(a)
		if err != nil {
			return domain.Attempt{}, err
		}
		timedOut = true
	}
	if !c.IsFinished() {
		return domain.Attempt{}, fmt.Errorf("%w: cannot determine execution result in container state %q", domain.ErrStartOutcomeUnknown, c.State)
	}
	if timedOut || a.DeadlineExceeded(c.FinishedAt) {
		_, err := a.FinishTimedOut(c.Exit.Code, time.Now())
		return a, err
	}
	var failure *string
	if c.HasError() {
		message := fmt.Sprintf("container exited with code %d", c.Exit.Code)
		if c.Exit.Error != "" {
			message += ": " + c.Exit.Error
		}
		failure = &message
	}
	if _, err := a.Finish(&c.Exit.Code, failure, time.Now()); err != nil {
		return domain.Attempt{}, err
	}
	return a, nil
}

func (r *PinnRunner) stopExpired(a domain.Attempt) (docker.Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	stopErr := r.docker.Stop(ctx, a.ContainerID, stopGracePeriod)
	// A failed request may have stopped the container before losing its response.
	// Reconcile with a read; never retry Stop or infer completion from its error.
	c, inspectErr := r.inspectAttempt(ctx, a.ContainerID, a)
	if inspectErr != nil {
		return docker.Container{}, fmt.Errorf("%w: confirm stopped container: %w", domain.ErrStopOutcomeUnknown, errors.Join(stopErr, inspectErr))
	}
	if !c.IsFinished() {
		stateErr := fmt.Errorf("exit not confirmed in state %q", c.State)
		return docker.Container{}, fmt.Errorf("%w: %w", domain.ErrStopOutcomeUnknown, errors.Join(stopErr, stateErr))
	}
	return c, nil
}
