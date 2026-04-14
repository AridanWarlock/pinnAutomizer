package server

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/middleware"
)

type ApiVersion int

type ApiVersionRouter struct {
	mux         *http.ServeMux
	apiVersion  ApiVersion
	middlewares []middleware.Middleware
}

func NewApiVersionRouter(apiVersion ApiVersion, middlewares ...middleware.Middleware) *ApiVersionRouter {
	return &ApiVersionRouter{
		mux:         http.NewServeMux(),
		apiVersion:  apiVersion,
		middlewares: middlewares,
	}
}

type HttpHandler interface {
	Route() Route
}

func (r *ApiVersionRouter) RegisterHandlers(handlers ...HttpHandler) {
	for _, handler := range handlers {
		r.registerRoute(handler.Route())
	}
}

func (r *ApiVersionRouter) RegisterRoutes(routes ...Route) {
	for _, route := range routes {
		r.registerRoute(route)
	}
}

func (r *ApiVersionRouter) registerRoute(route Route) {
	pattern := fmt.Sprintf("%s %s", route.Method, route.Path)

	r.mux.Handle(pattern, route.WithMiddleware())
}

func (r *ApiVersionRouter) WithMiddleware() http.Handler {
	return middleware.ChainMiddleware(r.mux, r.middlewares...)
}
