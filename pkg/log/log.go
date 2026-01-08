package log

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type Config struct {
	Level         string `default:"error" env:"LOGGER_LEVEL"`
	PrettyConsole bool   `default:"false" env:"LOGGER_PRETTY_CONSOLE"`
}

const (
	local = "local"
	dev   = "dev"
	prod  = "prod"
)

func New(env string, c Config) (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(c.Level)
	if err != nil {
		panic(fmt.Sprintf("unsupported level log: %s", c.Level))
	}
	zerolog.SetGlobalLevel(level)

	var logger zerolog.Logger
	switch env {
	case local:
		logger = zerolog.New(os.Stdout)
		zerolog.TimeFieldFormat = time.DateTime
	case dev:
		//logger = zerolog.New()
		//zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	default:
		return zerolog.Logger{}, fmt.Errorf("config: log: unsupported environment")
	}

	logger = logger.With().Timestamp().Logger()

	return logger, nil
}
