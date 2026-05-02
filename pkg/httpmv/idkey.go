package httpmv

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

const IdempotencyKeyHeader = "X-Idempotency-Key"

func IdKey() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := logger.FromContext(ctx)
			rh := httpout.NewHandler(w, log)

			idKeyString := r.Header.Get(IdempotencyKeyHeader)
			if idKeyString == "" {
				idKeyString = uuid.NewString()
			}

			idKey, err := core.NewIdempotencyKey(idKeyString)
			if err != nil {
				rh.ErrorResponse(
					fmt.Errorf("%w: parse idempotency key: %v", errs.ErrInvalidArgument, err),
					"failed to get idempotency key from headers",
				)
				return
			}

			ctx = idKey.WithContext(ctx)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
