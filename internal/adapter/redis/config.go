package redis

import "time"

type Config struct {
	Addr     string `env:"REDIS_ADDR" env-default:"redis:6379"`
	Password string `env:"REDIS_PASSWORD" env-default:""`
	DB       int    `env:"REDIS_DB" env-default:"0"`

	DialTimeout  time.Duration `env:"REDIS_DIAL_TIMEOUT" env-default:"5s"`
	ReadTimeout  time.Duration `env:"REDIS_READ_TIMEOUT" env-default:"3s"`
	WriteTimeout time.Duration `env:"REDIS_WRITE_TIMEOUT" env-default:"3s"`

	PoolSize           int           `env:"REDIS_POOL_SIZE" env-default:"10"`
	MinIdleConnections int           `env:"REDIS_MIN_IDLE_CONNECTIONS" env-default:"5"`
	PoolTimeout        time.Duration `env:"REDIS_POOL_TIMEOUT" env-default:"4s"`
}
