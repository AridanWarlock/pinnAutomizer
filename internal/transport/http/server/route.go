package httpServer

import (
	"net/http"

	httpMiddleware "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/middleware"
)

type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middlewares []httpMiddleware.Middleware
}

func (r Route) WithMiddleware() http.Handler {
	return httpMiddleware.ChainMiddleware(
		r.Handler,
		r.Middlewares...,
	)
}
