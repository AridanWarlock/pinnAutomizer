package sqlite

import "time"

type Config struct {
	MigrationsDir string        `env:"MIGRATIONS_DIR" envDefault:"migrations"`
	Path          string        `env:"PATH,required"`
	BusyTimeout   time.Duration `env:"BUSY_TIMEOUT" envDefault:"5s"`
}
