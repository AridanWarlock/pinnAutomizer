package httpsrv

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpmv"
)

type ApiVersion int

type ApiVersionRouter struct {
	mux         *http.ServeMux
	apiVersion  ApiVersion
	middlewares []httpmv.Middleware
}

func NewApiVersionRouter(apiVersion ApiVersion, middlewares ...httpmv.Middleware) *ApiVersionRouter {
	return &ApiVersionRouter{
		mux:         http.NewServeMux(),
		apiVersion:  apiVersion,
		middlewares: middlewares,
	}
}

func (r *ApiVersionRouter) RegisterHandlers(handlers ...HttpHandler) {
	for _, handler := range handlers {
		r.mux.Handle(handler.Pattern, handler.Handler)
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
	return httpmv.ChainMiddleware(r.mux, r.middlewares...)
}
