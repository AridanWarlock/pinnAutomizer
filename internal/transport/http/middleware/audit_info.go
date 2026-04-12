package httpMiddleware

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

const FingerprintHeader = "X-Fingerprint"

func AuditInfo() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			audit, err := auditInfoFromRequest(r)
			if err != nil {
				rh := httpResponse.NewHandler(w, logger.FromContext(ctx))
				rh.ErrorResponse(
					fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err),
					"failed on collect audit info",
				)
				return
			}

			ctx = audit.WithContext(ctx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func auditInfoFromRequest(r *http.Request) (domain.AuditInfo, error) {
	uaString := r.UserAgent()
	ua, err := domain.NewUserAgent(uaString)
	if err != nil {
		return domain.AuditInfo{}, err
	}

	ipString := rawIP(r)
	ip, err := domain.NewUserIP(ipString)
	if err != nil {
		return domain.AuditInfo{}, err
	}

	fpHash := rawFingerprintHash(r)
	fp, err := domain.NewFingerprintFromHash(fpHash)
	if err != nil {
		return domain.AuditInfo{}, err
	}

	return domain.NewAuditInfo(fp, ip, ua), nil
}

func rawFingerprintHash(r *http.Request) []byte {
	sb := strings.Builder{}
	sb.WriteString(r.UserAgent())
	sb.WriteString(r.Header.Get("Accept-Language"))
	sb.WriteString(r.Header.Get(FingerprintHeader))

	hash := sha256.Sum256([]byte(sb.String()))
	return hash[:]
}

func rawIP(r *http.Request) string {
	addresses := r.Header.Get("X-Forwarded-For")
	if addresses != "" {
		ip := strings.Split(addresses, ",")[0]
		return strings.TrimSpace(ip)
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
