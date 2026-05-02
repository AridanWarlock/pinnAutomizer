package httpsrv

import "net/http"

type HttpHandler struct {
	Pattern string
	Handler http.Handler
}

func (h *HttpHandler) Handle() (string, http.Handler) {
	return h.Pattern, h.Handler
}
