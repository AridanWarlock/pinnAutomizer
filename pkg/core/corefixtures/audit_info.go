package corefixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func NewAuditInfo(mods ...Mod[core.AuditInfo]) core.AuditInfo {
	audit := core.AuditInfo{
		Fingerprint: NewFingerprint(),
		IP:          NewUserIP(),
		Agent:       NewUserAgent(),
	}

	return Fixture(audit, mods)
}
