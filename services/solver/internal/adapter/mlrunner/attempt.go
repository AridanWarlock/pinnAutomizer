package mlrunner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

const (
	runLabel         = "pinn.solver.run_id"
	taskLabel        = "pinn.solver.task_id"
	modeLabel        = "pinn.solver.mode"
	reconcileTimeout = 5 * time.Second
)

// EnsureContainer may repeat Create by name, but never starts a computation.
func (r *PinnRunner) EnsureContainer(ctx context.Context, a domain.Attempt) (string, error) {
	return r.ensureContainer(ctx, a, []string{
		"python", "run.py", fmt.Sprintf("--mod=%s", a.Mode),
	})
}

func (r *PinnRunner) ensureContainer(ctx context.Context, a domain.Attempt, command []string) (string, error) {
	name := r.containerName(a.RunID)
	existing, err := r.inspectAttempt(ctx, name, a)
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, errs.ErrNotFound) {
		return "", err
	}

	opts := r.setupContainer(a.TaskID, a.Mode, command)
	opts.Name = name
	opts.Labels = map[string]string{
		runLabel:  a.RunID.String(),
		taskLabel: a.TaskID.String(),
		modeLabel: a.Mode,
	}
	opts.AutoRemove = false
	id, createErr := r.docker.Create(ctx, opts)
	if createErr == nil {
		return id, nil
	}

	// Create may have succeeded even when its response was lost or a competing
	// request won the name. Inspect with an independent bounded context.
	reconcile, cancel := context.WithTimeout(context.Background(), reconcileTimeout)
	defer cancel()
	existing, err = r.inspectAttempt(reconcile, name, a)
	if err != nil {
		return "", fmt.Errorf("reconcile container create: %w", errors.Join(createErr, err))
	}
	return existing.ID, nil
}

// StartContainer must only be called by the winner of persisted RequestStart.
// It never retries Start, including after a transport error or cancellation.
func (r *PinnRunner) StartContainer(ctx context.Context, a domain.Attempt) error {
	if !a.CanSendStart() {
		return fmt.Errorf("%w: attempt has no start intent", errs.ErrConflict)
	}
	ctx, cancel := context.WithDeadline(ctx, *a.Deadline)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	existing, err := r.inspectAttempt(ctx, a.ContainerID, a)
	if err != nil {
		return fmt.Errorf("inspect before start: %w", err)
	}
	if existing.HasStarted() {
		return nil
	}
	if !existing.IsCreated() {
		return fmt.Errorf("%w: container state %q", domain.ErrStartOutcomeUnknown, existing.State)
	}
	if err = r.docker.Start(ctx, a.ContainerID); err == nil {
		return nil
	}

	reconcile, stop := context.WithTimeout(context.Background(), reconcileTimeout)
	defer stop()
	existing, inspectErr := r.inspectAttempt(reconcile, a.ContainerID, a)
	if inspectErr == nil && existing.HasStarted() {
		return nil
	}
	return fmt.Errorf("%w: start container: %w", domain.ErrStartOutcomeUnknown, errors.Join(err, inspectErr))
}

func (r *PinnRunner) inspectAttempt(ctx context.Context, id string, a domain.Attempt) (docker.Container, error) {
	c, err := r.docker.Inspect(ctx, id)
	if err != nil {
		return docker.Container{}, fmt.Errorf("inspect attempt container: %w", err)
	}
	if !r.containerMatchAttempt(c, a) {
		return docker.Container{}, fmt.Errorf("%w: container does not match attempt", errs.ErrConflict)
	}
	return c, nil
}

func (r *PinnRunner) containerMatchAttempt(c docker.Container, a domain.Attempt) bool {
	return c.ID != "" && strings.TrimPrefix(c.Name, "/") == r.containerName(a.RunID) &&
		c.Labels[runLabel] == a.RunID.String() && c.Labels[taskLabel] == a.TaskID.String() &&
		c.Labels[modeLabel] == a.Mode && c.Image == r.pinnImage && !c.AutoRemove &&
		(c.RestartPolicy == "" || c.RestartPolicy == "no")
}
