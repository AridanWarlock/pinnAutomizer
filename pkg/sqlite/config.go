package sqlite

import "time"

type Config struct {
	Path        string        `env:"PATH,required"`
	BusyTimeout time.Duration `env:"BUSY_TIMEOUT" envDefault:"5s"`
}
