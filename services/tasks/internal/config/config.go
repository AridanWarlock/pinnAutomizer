package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/adapter/postgres"
)

type App struct {
	Env string `env:"ENVIRONMENT,required"`
}

type Config struct {
	App           App
	HTTP          httpsrv.Config       `envPrefix:"HTTP_"`
	Log           logger.Config        `envPrefix:"LOGGER_"`
	Postgres      postgres.Config      `envPrefix:"POSTGRES_"`
	KafkaProducer kafka.ProducerConfig `envPrefix:"KAFKA_PRODUCER_"`
	KafkaConsumer kafka.ConsumerConfig `envPrefix:"KAFKA_CONSUMER_"`
}

func InitConfig() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
