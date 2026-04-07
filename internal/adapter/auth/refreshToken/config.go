package refreshToken

import "time"

type Config struct {
	Ttl time.Duration `env:"TOKEN_TTL,required"`
}
