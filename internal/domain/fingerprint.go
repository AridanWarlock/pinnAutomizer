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
	if err := f.Validate(); err != nil {
		return nil, err
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
	if len(f) != 32 || bytes.Equal(f, zeroFingerprint) {
		return ErrInvalidFingerprint
	}
	return nil
}

func (f Fingerprint) Equal(other Fingerprint) bool {
	return subtle.ConstantTimeCompare(f, other) == 1
}
