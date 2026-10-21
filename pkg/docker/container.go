package docker

import (
	"fmt"
	"maps"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
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

type Exit struct {
	Code  int
	Error string
}

// ContainerInput is an untrusted snapshot used to restore a Container.
type ContainerInput struct {
	ID            string `validate:"required"`
	Name          string
	Image         string
	Labels        map[string]string
	AutoRemove    bool
	RestartPolicy string
	State         State     `validate:"required,oneof=created running paused restarting removing exited dead"`
	Running       bool      `validate:"required_if=State running,excluded_if=State exited"`
	StartedAt     time.Time `validate:"required_if=State running"`
	FinishedAt    time.Time `validate:"required_if=State exited"`
	Exit          *Exit     `validate:"required_if=State exited"`
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
	if in.Exit != nil {
		exit := *in.Exit
		c.Exit = &exit
	}
	return c, nil
}

func (c Container) Validate() error {
	if err := validate.V.Struct(c); err != nil {
		return fmt.Errorf("%w: %w", errs.ErrInvalidArgument, err)
	}
	if c.State == StateExited && (c.StartedAt.IsZero() || c.FinishedAt.Before(c.StartedAt)) {
		return fmt.Errorf("%w: container finish precedes its execution", errs.ErrInvalidArgument)
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

func (c Container) IsExited() bool {
	return c.State == StateExited
}

func (c Container) HasError() bool {
	return c.Exit != nil && c.Exit.HasError()
}

func (e Exit) HasError() bool {
	return e.Code != 0 || e.Error != ""
}
