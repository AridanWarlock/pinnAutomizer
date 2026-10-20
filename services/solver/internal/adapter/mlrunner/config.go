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

// containerName identifies an attempt independently of task IDs.
func (r *PinnRunner) containerName(runID uuid.UUID) string {
	prefix := r.containerNamePrefix
	if prefix == "" {
		prefix = DefaultContainerNamePrefix
	}
	return prefix + "-" + runID.String()
}
