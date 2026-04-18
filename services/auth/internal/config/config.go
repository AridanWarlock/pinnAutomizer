package config

import (
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jwt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/goRedis"
	"github.com/caarlos0/env/v11"
)

type App struct {
	Env string `env:"ENVIRONMENT,required"`
}

type Config struct {
	App                  App
	HTTP                 httpsrv.Config  `envPrefix:"HTTP_"`
	Log                  logger.Config   `envPrefix:"LOGGER_"`
	Postgres             postgres.Config `envPrefix:"POSTGRES_"`
	Redis                goRedis.Config  `envPrefix:"REDIS_"`
	AccessTokenGenerator jwt.Config      `envPrefix:"JWT_"`
}

func InitConfig() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
