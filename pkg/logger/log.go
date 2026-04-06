package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"

	LatencyTimeMeasurement = time.Millisecond
)

type ctxKey struct{}

func WithContext(ctx context.Context, log zerolog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

func FromContext(ctx context.Context) zerolog.Logger {
	log, ok := ctx.Value(ctxKey{}).(zerolog.Logger)
	if !ok {
		panic("no logger in context")
	}

	return log
}

func New(cfg Config) (zerolog.Logger, func(), error) {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		return zerolog.Logger{}, nil, fmt.Errorf("parse log level: %w", err)
	}
	zerolog.SetGlobalLevel(level)

	zerolog.DurationFieldInteger = false
	zerolog.DurationFieldUnit = LatencyTimeMeasurement

	consoleWriter, err := configureConsoleWriter(cfg.Env)
	if err != nil {
		return zerolog.Logger{}, nil, fmt.Errorf("configure console writer: %w", err)
	}
	fileWriter, cancel, err := configureFileWriter(cfg.Folder)
	if err != nil {
		return zerolog.Logger{}, nil, fmt.Errorf("configure file writer: %w", err)
	}

	logger := zerolog.New(zerolog.MultiLevelWriter(
		consoleWriter,
		fileWriter,
	)).With().
		Timestamp()

	switch cfg.Env {
	case envLocal, envDev:
		logger = logger.Caller()
	default:
	}

	return logger.Logger(), cancel, nil
}

func configureConsoleWriter(env string) (io.Writer, error) {
	var options func(w *zerolog.ConsoleWriter)

	switch env {
	case envLocal:
		options = func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = time.DateTime

			w.FormatFieldName = func(i interface{}) string {
				return fmt.Sprintf("\x1b[34m%s:\x1b[0m", i)
			}

			w.FormatFieldValue = func(i interface{}) string {
				return fmt.Sprintf("[%v]", i)
			}
		}
	case envDev, envProd:
		options = func(w *zerolog.ConsoleWriter) {
			w.TimeFormat = zerolog.TimeFieldFormat
		}
	default:
		return nil, fmt.Errorf("unsupported environment %s", env)
	}

	return zerolog.NewConsoleWriter(options), nil
}

func configureFileWriter(folder string) (io.Writer, func(), error) {
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, nil, fmt.Errorf("mkdir logger folder: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05.000000")
	logFilePath := filepath.Join(
		folder,
		fmt.Sprintf("%s.log", timestamp),
	)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	cancel := func() {
		if err := logFile.Close(); err != nil {
			fmt.Printf("closing log file error: %s\n", err.Error())
		}
	}

	return logFile, cancel, nil
}
