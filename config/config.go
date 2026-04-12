package config

import (
	"fmt"

	jwtToken "github.com/AridanWarlock/pinnAutomizer/internal/adapter/jwt/token"
	kafkaAtLeastOnceConsumer "github.com/AridanWarlock/pinnAutomizer/internal/adapter/kafkaConsumer/atLeastOnce"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/kafkaProducer"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/redis/goRedis"
	httpServer "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/caarlos0/env/v11"
)

type App struct {
	Env string `env:"ENVIRONMENT,required"`
}

type Config struct {
	App                  App
	HTTP                 httpServer.Config               `envPrefix:"HTTP_"`
	Log                  logger.Config                   `envPrefix:"LOGGER_"`
	Postgres             postgres.Config                 `envPrefix:"POSTGRES_"`
	Redis                goRedis.Config                  `envPrefix:"REDIS_"`
	AccessTokenGenerator jwtToken.Config                 `envPrefix:"JWT_"`
	KafkaProducer        kafkaProducer.Config            `envPrefix:"KAFKA_PRODUCER_"`
	KafkaConsumer        kafkaAtLeastOnceConsumer.Config `envPrefix:"KAFKA_CONSUMER_"`
}

func InitConfig() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
