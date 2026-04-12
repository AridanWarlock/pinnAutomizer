package httpMiddleware

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

func Recover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := logger.FromContext(r.Context())
			handler := httpResponse.NewHandler(w, log)

			log.Debug().Msg(fmt.Sprintf("get audit info: %v", domain.AuditInfoFromContext(r.Context())))

			defer func() {
				if p := recover(); p != nil {
					handler.PanicResponse(p, "during handle HTTP request got unexpected panic")
				}
			}()

			next.ServeHTTP(w, r)
			log.Debug().Msg(fmt.Sprintf("get audit info: %v", domain.AuditInfoFromContext(r.Context())))

		})
	}
}
