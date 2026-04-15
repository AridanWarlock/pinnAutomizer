package core

import (
	"encoding/hex"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Fingerprint string

func NewFingerprintFromHash(hash []byte) (Fingerprint, error) {
	return NewFingerprint(hex.EncodeToString(hash))
}

func NewFingerprint(hex string) (Fingerprint, error) {
	f := Fingerprint(hex)
	if err := f.Validate(); err != nil {
		return "", err
	}
	return f, nil
}

func (f Fingerprint) Validate() error {
	err := validate.V.Var(
		f.String(),
		"required,hexadecimal,len=64",
	)
	if err != nil {
		return ErrInvalidFingerprint
	}
	return nil
}

func (f Fingerprint) String() string {
	return string(f)
}
