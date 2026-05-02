package mlrunner

import "time"

type Config struct {
	Image string `env:"IMAGE"`

	Timeout time.Duration `env:"TIMEOUT,required"`
}
