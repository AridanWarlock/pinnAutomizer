package httpmv

import (
	"net/http"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

func TraceID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.FromContext(r.Context())
			rw := httpout.NewResponseWriter(w)

			log.Debug().
				Msg(">>> incoming HTTP request")

			start := time.Now()
			next.ServeHTTP(rw, r)
			duration := time.Since(start)

			log.Debug().
				Dur("latency", duration).
				Int("status_code", rw.GetStatusCode()).
				Msg("<<< done HTTP request")
		})
	}
}
