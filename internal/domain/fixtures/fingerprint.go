package fixtures

import (
	"encoding/hex"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
)

func NewFingerprint(mods ...mod[domain.Fingerprint]) domain.Fingerprint {
	bytes, _ := hex.DecodeString("01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b")
	f := domain.Fingerprint(bytes)

	return fixture(f, mods)
}
