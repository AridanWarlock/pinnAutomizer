package postgres

import "time"

type Config struct {
	Addr    string        `env:"POSTGRES_URL" env-required:"true"`
	Timeout time.Duration `env:"POSTGRES_TIMEOUT" env-required:"true"`
}
