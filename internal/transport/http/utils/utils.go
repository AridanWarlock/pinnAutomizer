package http_utils

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/middleware"
)

func ClaimsFromContext(ctx context.Context) domain.UserClaims {
	claims, ok := ctx.Value(http_middleware.UserClaimsKey).(domain.UserClaims)
	if !ok {
		panic("no user claims in context")
	}

	return claims
}
