package logger

import (
	"log/slog"
	"os"
)

type Config struct {
	Level         string `default:"error" env:"LOGGER_LEVEL"`
	PrettyConsole bool   `default:"false" env:"LOGGER_PRETTY_CONSOLE"`
}

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func New(env string, c Config) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
