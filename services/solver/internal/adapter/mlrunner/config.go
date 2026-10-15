package mlrunner

import (
	"time"

	"github.com/google/uuid"
)

const DefaultContainerNamePrefix = "pinn-solver"

type Config struct {
	ContainerNamePrefix string        `env:"CONTAINER_NAME_PREFIX" envDefault:"pinn-solver"`
	Image               string        `env:"IMAGE,required"`
	HostTasksDataDir    string        `env:"HOST_TASKS_DATA_DIR,required"`
	HostTasksOutputDir  string        `env:"HOST_TASKS_OUTPUT_DIR,required"`
	Timeout             time.Duration `env:"TIMEOUT,required"`
}

// ContainerName identifies an attempt for future Docker recovery, independently of task IDs.
func (c Config) ContainerName(runID uuid.UUID) string {
	prefix := c.ContainerNamePrefix
	if prefix == "" {
		prefix = DefaultContainerNamePrefix
	}
	return prefix + "-" + runID.String()
}
