package proxy

import (
	"context"
	"fmt"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

type ServiceProxy struct {
	*httputil.ReverseProxy
}

func NewServiceProxy(targetHost string) (*ServiceProxy, error) {
	targetUrl, err := url.Parse(targetHost)
	if err != nil {
		return nil, fmt.Errorf("parse url from target host: %w", err)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(preq *httputil.ProxyRequest) {
			clearMaliciousHeaders(preq)

			injectToHeaders(preq, getInjectItems(preq.In.Context()))

			preq.SetXForwarded()
			preq.SetURL(targetUrl)
		},
	}

	return &ServiceProxy{proxy}, nil
}

func clearMaliciousHeaders(r *httputil.ProxyRequest) {
	for header := range r.Out.Header {
		if strings.HasPrefix(header, "X-Internal-") {
			r.Out.Header.Del(header)
		}
	}
}

func injectToHeaders(preq *httputil.ProxyRequest, items []core.ToHeadersSerializable) {
	for _, item := range items {
		for key, val := range item.ToHeaders() {
			preq.Out.Header.Set(key, val)
		}
	}
}

func getInjectItems(ctx context.Context) []core.ToHeadersSerializable {
	items := []core.ToHeadersSerializable{
		core.MustAuditInfoFromContext(ctx),
	}

	if auth, ok := core.AuthInfoFromContext(ctx); ok {
		items = append(items, auth)
	}

	return items
}
