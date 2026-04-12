package domain

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

func AuditInfoFromContext(ctx context.Context) AuditInfo {
	v, ok := ctx.Value(auditInfoKey{}).(AuditInfo)
	if !ok {
		panic("no audit info in context")
	}
	return v
}
