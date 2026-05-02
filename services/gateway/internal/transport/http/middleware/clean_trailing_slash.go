package middleware

import (
	"net/http"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpmv"
)

func CleanTrailingSlash() httpmv.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" && strings.HasSuffix(r.URL.Path, "/") {
				r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
			}

			next.ServeHTTP(w, r)
		})
	}
}
