package server

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/middleware"
)

type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middlewares []middleware.Middleware
}

func (r Route) WithMiddleware() http.Handler {
	return middleware.ChainMiddleware(
		r.Handler,
		r.Middlewares...,
	)
}
