package fixtures

import "github.com/AridanWarlock/pinnAutomizer/internal/domain"

func NewAuditInfo(mods ...mod[domain.AuditInfo]) domain.AuditInfo {
	audit := domain.AuditInfo{
		Fingerprint: NewFingerprint(),
		IP:          NewUserIP(),
		Agent:       NewUserAgent(),
	}

	return fixture(audit, mods)
}
