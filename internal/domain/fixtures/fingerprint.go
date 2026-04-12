package fixtures

import (
	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

func NewFingerprint(mods ...mod[domain.Fingerprint]) domain.Fingerprint {
	f := domain.Fingerprint("01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b")

	return fixture(f, mods)
}
