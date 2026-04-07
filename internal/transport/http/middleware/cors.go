package httpMiddleware

import (
	"net/http"
	"strconv"
	"strings"
)

func Cors() Middleware {
	allowedOrigins := map[string]struct{}{
		"http://localhost:8080": {},
		"http://0.0.0.0:8080":   {},
	}

	allowedMethods := strings.Join([]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPatch,
		http.MethodPut,
		http.MethodDelete,
		http.MethodOptions,
	}, ", ")

	allowedHeaders := strings.Join([]string{
		"Content-Type",
		"Authorization",
		"Accept",
		"Origin",
		"X-Requested-With",
		"X-CSRF-Token",
		"X-Fingerprint",

		"Content-Disposition",
		"Content-Length",
	}, ", ")

	exposedHeaders := strings.Join([]string{
		"Content-Length",
		"Content-Type",
		"Authorization",
		"Set-Cookie",
	}, ", ")

	allowCredentials := true

	maxAge := 300

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if _, ok := allowedOrigins[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
				w.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(allowCredentials))
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
