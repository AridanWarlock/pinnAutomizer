package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func NewAuditInfo(mods ...mod[core.AuditInfo]) core.AuditInfo {
	audit := core.AuditInfo{
		Fingerprint: NewFingerprint(),
		IP:          NewUserIP(),
		Agent:       NewUserAgent(),
	}

	return fixture(audit, mods)
}
