package core_http_server

import (
	"fmt"
	"net/http"
)

type ApiVersion string

var (
	ApiVersion1 = ApiVersion("v1")
)

type ApiVersionRouter struct {
	*http.ServeMux
	apiVersion ApiVersion
}

func NewApiVersionRouter(apiVersion ApiVersion) *ApiVersionRouter {
	return &ApiVersionRouter{
		ServeMux:   http.NewServeMux(),
		apiVersion: apiVersion,
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

	r.Handle(pattern, route.Handler)
}
