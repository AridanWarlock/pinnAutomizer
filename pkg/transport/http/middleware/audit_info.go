package middleware

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/response"
)

func AuditInfo() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			audit, err := auditInfoFromHeaders(r.Header)
			if err == nil {
				ctx = audit.WithContext(ctx)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			log := logger.FromContext(ctx)
			rh := response.NewHandler(w, log)
			rh.ErrorResponse(
				fmt.Errorf("parse audit info from headers: %w", err),
				"failed on parse audit info from headers",
			)
		})
	}
}

func auditInfoFromHeaders(headers http.Header) (core.AuditInfo, error) {
	fp, err := core.NewFingerprint(headers.Get(core.FingerprintHeader))
	if err != nil {
		return core.AuditInfo{}, fmt.Errorf("parse fingerprint from headers: %w", err)
	}

	ip, err := core.NewUserIP(headers.Get(core.UserIPHeader))
	if err != nil {
		return core.AuditInfo{}, fmt.Errorf("parse ip from headers: %w", err)
	}

	agent, err := core.NewUserAgent(headers.Get(core.UserAgentHeader))
	if err != nil {
		return core.AuditInfo{}, fmt.Errorf("parse ip from headers: %w", err)
	}

	return core.NewAuditInfo(fp, ip, agent), nil
}
