package http_middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const requestIDHeader = "X-Request-ID"

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = uuid.NewString()
			}

			r.Header.Set(requestIDHeader, requestID)
			w.Header().Set(requestIDHeader, requestID)

			next.ServeHTTP(w, r)
		})
	}
}

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

			ctx := context.WithValue(r.Context(), logger.ContextKey, log)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Recover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.FromContext(r.Context())
			handler := http_response.NewHandler(w, log)

			defer func() {
				if p := recover(); p != nil {
					handler.PanicResponse(p, "during handle HTTP request got unexpected panic")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func TraceID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.FromContext(r.Context())
			rw := http_response.NewResponseWriter(w)

			log.Debug().
				Msg(">>> incoming HTTP request")

			start := time.Now()
			next.ServeHTTP(rw, r)
			duration := time.Since(start)

			log.Debug().
				Dur("latency", duration).
				Int("status_code", rw.GetStatusCodeOrPanic()).
				Msg("<<< done HTTP request")
		})
	}
}
