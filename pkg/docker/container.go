package docker

import "time"

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

// Container is an infrastructure snapshot returned by Inspect.
type Container struct {
	ID            string
	Name          string
	Image         string
	Labels        map[string]string
	AutoRemove    bool
	RestartPolicy string
	State         string
	Running       bool
	StartedAt     time.Time
	FinishedAt    time.Time
	ExitCode      int
	Error         string
}

func (c Container) IsCreated() bool {
	return c.State == "created"
}

func (c Container) HasStarted() bool {
	return c.Running || !c.StartedAt.IsZero()
}

type Exit struct {
	Code  int
	Error string
}

func (c Container) IsExited() bool {
	return c.State == "exited" && !c.Running
}
