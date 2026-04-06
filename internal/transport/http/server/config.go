package core_http_server

import "time"

type Config struct {
	Addr            string        `env:"HTTP_ADDR" env-required:"true"`
	Timeout         time.Duration `env:"HTTP_TIMEOUT" env-required:"true"`
	IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT" env-required:"true"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-required:"true"`
}
