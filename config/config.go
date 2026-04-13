package config

import (
	"fmt"

	jwtToken "github.com/AridanWarlock/pinnAutomizer/internal/adapter/jwt/token"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/goRedis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/server"
	"github.com/caarlos0/env/v11"
)

type App struct {
	Env string `env:"ENVIRONMENT,required"`
}

type Config struct {
	App                  App
	HTTP                 server.Config      `envPrefix:"HTTP_"`
	Log                  logger.Config      `envPrefix:"LOGGER_"`
	Postgres             postgres.Config    `envPrefix:"POSTGRES_"`
	Redis                goRedis.Config     `envPrefix:"REDIS_"`
	AccessTokenGenerator jwtToken.Config    `envPrefix:"JWT_"`
	KafkaWriter          kafka.WriterConfig `envPrefix:"KAFKA_WRITER_"`
	KafkaReader          kafka.ReaderConfig `envPrefix:"KAFKA_READER_"`
}

func InitConfig() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
