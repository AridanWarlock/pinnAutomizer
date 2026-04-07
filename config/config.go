package config

import (
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/kafka_produce"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/redis"
	http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/auth/jwt/access_token"
	"github.com/AridanWarlock/pinnAutomizer/pkg/auth/refresh_token"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/caarlos0/env/v11"
)

type App struct {
	Env string `env:"ENVIRONMENT,required"`
}

type Config struct {
	App                   App
	HTTP                  http_server.Config   `envPrefix:"HTTP_"`
	Log                   logger.Config        `envPrefix:"LOGGER_"`
	Postgres              postgres.Config      `envPrefix:"POSTGRES_"`
	Redis                 redis.Config         `envPrefix:"REDIS_"`
	AccessTokenGenerator  access_token.Config  `envPrefix:"JWT_"`
	RefreshTokenGenerator refresh_token.Config `envPrefix:"REFRESH_"`
	KafkaProducer         kafka_produce.Config `envPrefix:"KAFKA_PRODUCER_"`
	//KafkaConsumerOnTrain    update_task_status_on_train.Config
	//KafkaConsumerAfterTrain update_task_status_after_train.Config
}

func InitConfig() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
