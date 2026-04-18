package httpsrv

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpmv"
)

type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middlewares []httpmv.Middleware
	IsPublic    bool
}

func (r Route) WithMiddleware() http.Handler {
	if r.IsPublic {
		return httpmv.ChainMiddleware(
			r.Handler,
			r.Middlewares...,
		)
	}
	return httpmv.ChainMiddleware(
		r.Handler,
		append(r.Middlewares, httpmv.AuthInfo())...,
	)
}
