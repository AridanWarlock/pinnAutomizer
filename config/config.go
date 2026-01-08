package config

import (
	"pinnAutomizer/internal/adapter/postgres"
	"pinnAutomizer/internal/adapter/translator"
	"pinnAutomizer/pkg/httpserver"
	"pinnAutomizer/pkg/jwt"
	"pinnAutomizer/pkg/log"

	"github.com/ilyakaznacheev/cleanenv"
)

type App struct {
	Name    string `env:"APP_NAME" required:"true"`
	Version string `env:"APP_VERSION" required:"true"`
	Env     string `env:"ENVIRONMENT" required:"true"`
}

type Config struct {
	App        App
	HTTP       httpserver.Config
	Log        log.Config
	Postgres   postgres.Config
	Translator translator.Config
	Jwt        jwt.Config
}

func InitConfig() (Config, error) {
	c := Config{}
	err := cleanenv.ReadEnv(&c)

	return c, err
}
