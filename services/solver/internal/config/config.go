package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
)

type App struct {
	Env string `env:"ENVIRONMENT,required"`
}

type Recovery struct {
	ActiveLimit    int           `env:"ACTIVE_LIMIT" envDefault:"1" validate:"gt=0"`
	BatchSize      int           `env:"BATCH_SIZE" envDefault:"8" validate:"gt=0"`
	Interval       time.Duration `env:"INTERVAL" envDefault:"1s" validate:"gt=0"`
	AttemptTimeout time.Duration `env:"ATTEMPT_TIMEOUT" envDefault:"10s" validate:"gt=0"`
}

type Config struct {
	App            App
	Recovery       Recovery             `envPrefix:"RECOVERY_"`
	Log            logger.Config        `envPrefix:"LOGGER_"`
	PinnRunner     mlrunner.Config      `envPrefix:"PINN_RUNNER_"`
	KafkaProducer  kafka.ProducerConfig `envPrefix:"KAFKA_PRODUCER_"`
	KafkaConsumer  kafka.ConsumerConfig `envPrefix:"KAFKA_CONSUMER_"`
	SQLite         sqlite.Config        `envPrefix:"SQLITE_"`
	OutboxInterval time.Duration        `env:"OUTBOX_INTERVAL" envDefault:"1s" validate:"gt=0"`
}

func InitConfig() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if err := validate.V.Struct(c); err != nil {
		return fmt.Errorf("%w: solver config: %w", errs.ErrInvalidArgument, err)
	}
	if c.PinnRunner.Timeout <= 0 {
		return fmt.Errorf("%w: positive PINN timeout required", errs.ErrInvalidArgument)
	}
	return nil
}
