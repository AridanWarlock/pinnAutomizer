package config

import (
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/caarlos0/env/v11"
)

type App struct {
	Env string `env:"ENVIRONMENT,required"`
}

type Config struct {
	App         App
	Log         logger.Config      `envPrefix:"LOGGER_"`
	PinnRunner  mlrunner.Config    `envPrefix:"PINN_RUNNER_"`
	KafkaWriter kafka.WriterConfig `envPrefix:"KAFKA_WRITER_"`
	KafkaReader kafka.ReaderConfig `envPrefix:"KAFKA_READER_"`
}

func InitConfig() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
