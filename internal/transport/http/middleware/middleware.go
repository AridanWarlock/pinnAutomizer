package http_middleware

import (
	"net/http"
	"slices"
)

type Middleware func(next http.Handler) http.Handler

func ChainMiddleware(
	handler http.Handler,
	middlewares ...Middleware,
) http.Handler {
	for _, middleware := range slices.Backward(middlewares) {
		handler = middleware(handler)
	}

	return handler
}
