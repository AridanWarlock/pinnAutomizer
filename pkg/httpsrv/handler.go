package httpsrv

import "net/http"

type HttpHandler struct {
	Pattern string
	Handler http.Handler
}
