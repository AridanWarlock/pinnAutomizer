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
	IsPublic    bool
}

func (r Route) WithMiddleware() http.Handler {
	if r.IsPublic {
		return middleware.ChainMiddleware(
			r.Handler,
			r.Middlewares...,
		)
	}
	return middleware.ChainMiddleware(
		r.Handler,
		append(r.Middlewares, middleware.AuthInfo())...,
	)
}
