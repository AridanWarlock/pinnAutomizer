package server

import "time"

type Config struct {
	Addr            string        `env:"ADDR,required"`
	Timeout         time.Duration `env:"TIMEOUT,required"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT,required"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required"`
}
