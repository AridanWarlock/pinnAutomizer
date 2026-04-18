package httpmv

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/rs/zerolog"
)

func Logger(log zerolog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				panic("initialize middleware: logger middleware: zero request id header")
			}

			log = log.With().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Logger()

			ctx := logger.WithContext(r.Context(), log)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
