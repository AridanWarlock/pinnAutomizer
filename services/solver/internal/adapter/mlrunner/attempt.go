package mlrunner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

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
	name := r.attemptName(a)
	existing, err := r.inspectAttempt(ctx, name, a)
	if err == nil {
		return existing.ID, nil
	}
	if !errdefs.IsNotFound(err) {
		return "", err
	}

	opts := r.setupContainer(a.TaskID, a.Mode, command)
	opts.Name = name
	opts.Config.Labels = map[string]string{
		runLabel:  a.RunID.String(),
		taskLabel: a.TaskID.String(),
		modeLabel: a.Mode,
	}
	opts.HostConfig.AutoRemove = false
	resp, createErr := r.cli.ContainerCreate(ctx, opts)
	if createErr == nil {
		return resp.ID, nil
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
	if a.State != domain.AttemptStartRequested {
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
	if hasStarted(existing.State) {
		return nil
	}
	if existing.State.Status != container.StateCreated {
		return fmt.Errorf("%w: container state %q", domain.ErrStartOutcomeUnknown, existing.State.Status)
	}
	if _, err = r.cli.ContainerStart(ctx, a.ContainerID, client.ContainerStartOptions{}); err == nil {
		return nil
	}

	reconcile, stop := context.WithTimeout(context.Background(), reconcileTimeout)
	defer stop()
	existing, inspectErr := r.inspectAttempt(reconcile, a.ContainerID, a)
	if inspectErr == nil && hasStarted(existing.State) {
		return nil
	}
	return fmt.Errorf("%w: start container: %w", domain.ErrStartOutcomeUnknown, errors.Join(err, inspectErr))
}

func (r *PinnRunner) attemptName(a domain.Attempt) string {
	return (Config{ContainerNamePrefix: r.containerNamePrefix}).ContainerName(a.RunID)
}

func (r *PinnRunner) inspectAttempt(ctx context.Context, id string, a domain.Attempt) (container.InspectResponse, error) {
	resp, err := r.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, fmt.Errorf("inspect attempt container: %w", err)
	}
	c := resp.Container
	if !r.containerMatchAttempt(c, a) {
		return container.InspectResponse{}, fmt.Errorf("%w: container does not match attempt", errs.ErrConflict)
	}
	return c, nil
}

func (r *PinnRunner) containerMatchAttempt(c container.InspectResponse, a domain.Attempt) bool {
	return c.ID != "" && c.Config != nil && c.HostConfig != nil && c.State != nil &&
		strings.TrimPrefix(c.Name, "/") == r.attemptName(a) &&
		c.Config.Labels[runLabel] == a.RunID.String() &&
		c.Config.Labels[taskLabel] == a.TaskID.String() &&
		c.Config.Labels[modeLabel] == a.Mode &&
		c.Config.Image == r.pinnImage && c.HostConfig.AutoRemove &&
		(c.HostConfig.RestartPolicy.Name == "" || c.HostConfig.RestartPolicy.Name == container.RestartPolicyDisabled)
}

func hasStarted(state *container.State) bool {
	// Exited containers must never be restarted. A non-zero StartedAt is evidence
	// of execution even if the command completed before we could observe running.
	if state.Running {
		return true
	}
	started, err := time.Parse(time.RFC3339Nano, state.StartedAt)
	return err == nil && !started.IsZero()
}
