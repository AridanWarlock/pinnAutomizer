package docker

import (
	"fmt"
	"maps"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type State string

const (
	StateCreated    State = "created"
	StateRunning    State = "running"
	StatePaused     State = "paused"
	StateRestarting State = "restarting"
	StateRemoving   State = "removing"
	StateExited     State = "exited"
	StateDead       State = "dead"
)

// Config is Docker create input, not a domain entity.
type Config struct {
	Name           string
	Image          string
	Command        []string
	Env            []string
	Labels         map[string]string
	Mounts         []Mount
	GPU            bool
	AutoRemove     bool
	ReadonlyRootfs bool
	NetworkMode    string
	SecurityOpt    []string
	CapDrop        []string
}

type Mount struct {
	Source     string
	Target     string
	ReadOnly   bool
	TmpfsBytes int64
	TmpfsMode  uint32
}

// ContainerInput is an untrusted snapshot used to restore a Container.
type ContainerInput struct {
	ID            string
	Name          string
	Image         string
	Labels        map[string]string
	AutoRemove    bool
	RestartPolicy string
	State         State
	Running       bool
	StartedAt     time.Time
	FinishedAt    time.Time
	ExitCode      int
	Error         string
}

// Container represents a validated Docker container snapshot.
// Exported fields follow the existing API convention; callers must not mutate them.
type Container ContainerInput

func NewContainer(in ContainerInput) (Container, error) {
	c := Container(in)
	if err := c.Validate(); err != nil {
		return Container{}, err
	}
	c.Labels = maps.Clone(in.Labels)
	return c, nil
}

func (c Container) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("%w: container ID is empty", errs.ErrInvalidArgument)
	}
	switch c.State {
	case StateCreated, StateRunning, StatePaused, StateRestarting, StateRemoving, StateExited, StateDead:
	default:
		return fmt.Errorf("%w: unknown container state %q", errs.ErrInvalidArgument, c.State)
	}
	if c.State == StateRunning && (!c.Running || c.StartedAt.IsZero()) {
		return fmt.Errorf("%w: running container has no active execution", errs.ErrInvalidArgument)
	}
	if c.State == StateExited && (c.Running || c.StartedAt.IsZero() || c.FinishedAt.IsZero() || c.FinishedAt.Before(c.StartedAt)) {
		return fmt.Errorf("%w: exited container has no completed execution", errs.ErrInvalidArgument)
	}
	return nil
}

func (c Container) IsCreated() bool {
	return c.State == StateCreated
}

func (c Container) IsRunning() bool {
	return c.Running
}

func (c Container) HasStarted() bool {
	return c.Running || !c.StartedAt.IsZero()
}

// IsFinished reports a confirmed execution result, unlike dead/removing states.
func (c Container) IsFinished() bool {
	return c.State == StateExited
}

type Exit struct {
	Code  int
	Error string
}

func (c Container) IsExited() bool {
	return c.State == StateExited
}
