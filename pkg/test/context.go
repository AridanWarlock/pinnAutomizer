package test

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/rs/zerolog"
)

func ContextWithZeroLogger() context.Context {
	return logger.WithContext(context.Background(), zerolog.Nop())
}

func ContextWithAuditInfo(ctx context.Context, info core.AuditInfo) context.Context {
	return info.WithContext(ctx)
}

func SetUpContext(audit core.AuditInfo, auth domain.AuthInfo) context.Context {
	ctx := ContextWithZeroLogger()
	ctx = audit.WithContext(ctx)
	ctx = auth.WithContext(ctx)

	return ctx
}
