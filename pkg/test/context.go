package test

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	httpMiddleware "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/middleware"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/rs/zerolog"
)

func ContextBackgroundWithZeroLogger() context.Context {
	return logger.WithContext(context.Background(), zerolog.Nop())
}

func ContextWithUserClaims(ctx context.Context, c domain.UserClaims) context.Context {
	return context.WithValue(ctx, httpMiddleware.UserClaimsKey, c)
}
