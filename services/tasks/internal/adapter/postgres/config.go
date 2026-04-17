package postgres

import "time"

type Config struct {
	User     string        `env:"USER,required"`
	Password string        `env:"PASSWORD,required"`
	Host     string        `env:"HOST,required"`
	Port     int           `env:"PORT,required"`
	DB       string        `env:"DB,required"`
	SslMode  string        `env:"SSLMODE" default:"disable"`
	Timeout  time.Duration `env:"TIMEOUT,required"`
}
