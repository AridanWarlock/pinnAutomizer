package httpsrv

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpmv"
)

type Route struct {
	Method             string
	Path               string
	Handler            http.HandlerFunc
	Middlewares        []httpmv.Middleware
	IsPublic           bool
	NeedIdempotencyKey bool
}

func (r Route) WithMiddleware() http.Handler {
	middlewares := r.Middlewares

	if r.NeedIdempotencyKey {
		middlewares = append(middlewares, httpmv.IdKey())
	}
	if !r.IsPublic {
		middlewares = append(middlewares, httpmv.AuthInfo())
	}

	return httpmv.ChainMiddleware(
		r.Handler,
		middlewares...,
	)
}
