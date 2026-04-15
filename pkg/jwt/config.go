package jwt

type Config struct {
	Secret string `env:"SECRET,required"`
}
