package core

import "context"

type auditInfoKey struct{}

type AuditInfo struct {
	Fingerprint Fingerprint
	IP          UserIP
	Agent       UserAgent
}

func NewAuditInfo(
	fingerprint Fingerprint,
	ip UserIP,
	agent UserAgent,
) AuditInfo {
	return AuditInfo{
		Fingerprint: fingerprint,
		IP:          ip,
		Agent:       agent,
	}
}

func (a AuditInfo) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, auditInfoKey{}, a)
}

func AuditInfoFromContext(ctx context.Context) (AuditInfo, bool) {
	v, ok := ctx.Value(auditInfoKey{}).(AuditInfo)
	return v, ok
}

func MustAuditInfoFromContext(ctx context.Context) AuditInfo {
	v, ok := AuditInfoFromContext(ctx)
	if !ok {
		panic("no audit info in context")
	}
	return v
}

func (a AuditInfo) ToHeaders() map[string]string {
	return map[string]string{
		UserIPHeader:      a.IP.String(),
		UserAgentHeader:   a.Agent.String(),
		FingerprintHeader: a.Fingerprint.String(),
	}
}
