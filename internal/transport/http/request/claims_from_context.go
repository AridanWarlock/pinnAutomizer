package httpRequest

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	httpMiddleware "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/middleware"
)

func ClaimsFromContext(ctx context.Context) domain.UserClaims {
	claims, ok := ctx.Value(httpMiddleware.UserClaimsKey).(domain.UserClaims)
	if !ok {
		panic("no user claims in context")
	}

	return claims
}
