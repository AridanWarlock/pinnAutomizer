package sqlite

import "time"

type Config struct {
	Timeout     time.Duration `env:"TIMEOUT" envDefault:"5s"`
	Path        string        `env:"PATH,required"`
	BusyTimeout time.Duration `env:"BUSY_TIMEOUT" envDefault:"5s"`
}
