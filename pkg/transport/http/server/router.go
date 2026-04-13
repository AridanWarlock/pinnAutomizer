package server

import (
	"fmt"
	"net/http"

	httpMiddleware "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/middleware"
)

type ApiVersion int

type ApiVersionRouter struct {
	mux         *http.ServeMux
	apiVersion  ApiVersion
	middlewares []httpMiddleware.Middleware
}

func NewApiVersionRouter(apiVersion ApiVersion, middlewares ...httpMiddleware.Middleware) *ApiVersionRouter {
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
	return httpMiddleware.ChainMiddleware(r.mux, r.middlewares...)
}
