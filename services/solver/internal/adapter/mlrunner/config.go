package mlrunner

import "time"

type Config struct {
	Image             string        `env:"IMAGE,required"`
	TasksDataVolume   string        `env:"TASKS_DATA_VOLUME,required"`
	TasksOutputVolume string        `env:"TASKS_OUTPUT_VOLUME,required"`
	Timeout           time.Duration `env:"TIMEOUT,required"`
}
