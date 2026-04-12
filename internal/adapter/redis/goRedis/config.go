package goRedis

import "time"

type Config struct {
	Addr     string `env:"ADDR,required"`
	Password string `env:"PASSWORD,required"`
	DB       int    `env:"DB,required"`

	DialTimeout  time.Duration `env:"DIAL_TIMEOUT,required"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT,required"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT,required"`

	PoolSize           int           `env:"POOL_SIZE,required"`
	MinIdleConnections int           `env:"MIN_IDLE_CONNECTIONS,required"`
	PoolTimeout        time.Duration `env:"POOL_TIMEOUT,required"`
}
