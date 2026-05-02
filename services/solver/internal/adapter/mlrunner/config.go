package mlrunner

import "time"

type Config struct {
	Image              string        `env:"IMAGE,required"`
	HostTasksDataDir   string        `env:"HOST_TASKS_DATA_DIR,required"`
	HostTasksOutputDir string        `env:"HOST_TASKS_OUTPUT_DIR,required"`
	Timeout            time.Duration `env:"TIMEOUT,required"`
}
