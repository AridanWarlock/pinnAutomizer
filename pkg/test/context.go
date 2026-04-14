package test

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/rs/zerolog"
)

func ContextWithZeroLogger() context.Context {
	return logger.WithContext(context.Background(), zerolog.Nop())
}
