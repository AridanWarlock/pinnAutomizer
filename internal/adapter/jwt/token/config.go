package jwtToken

type Config struct {
	Secret string `env:"SECRET,required"`
}
