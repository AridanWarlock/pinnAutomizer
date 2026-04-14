package core

import (
	"encoding/hex"
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
	if len(f) != 64 {
		return ErrInvalidFingerprint
	}
	return nil
}

func (f Fingerprint) String() string {
	return string(f)
}
