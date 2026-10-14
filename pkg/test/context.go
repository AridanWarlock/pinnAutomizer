package test

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

func ContextWithZeroLogger() context.Context {
	return logger.WithContext(context.Background(), zerolog.Nop())
}
