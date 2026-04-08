package domain

import (
	"bytes"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

var zeroFingerprint = Fingerprint(make([]byte, 32))

type Fingerprint []byte

func NewFingerprint(hash []byte) (Fingerprint, error) {
	f := Fingerprint(hash)
	if !f.IsValid() {
		return nil, ErrInvalidFingerprint
	}
	return f, nil
}

func NewFingerprintFromHex(s string) (Fingerprint, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return Fingerprint{}, fmt.Errorf("decode hex: %w", ErrInvalidFingerprint)
	}
	return NewFingerprint(b)
}

func (f Fingerprint) Validate() error {
	if !f.IsValid() {
		return ErrInvalidFingerprint
	}
	return nil
}

func (f Fingerprint) IsValid() bool {
	return len(f) == 32 && !bytes.Equal(f, zeroFingerprint)
}

func (f Fingerprint) Equal(other Fingerprint) bool {
	return subtle.ConstantTimeCompare(f, other) == 1
}
