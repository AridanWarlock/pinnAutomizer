package jwtToken

import "time"

type Config struct {
	Secret   string        `env:"SECRET,required"`
	TokenTTL time.Duration `env:"TOKEN_TTL,required"`
}
