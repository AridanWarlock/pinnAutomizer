package crypt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func GenerateSecureToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)

	return base64.RawURLEncoding.EncodeToString(b)
}

func Sha256(token string) string {
	b := sha256.Sum256([]byte(token))

	return hex.EncodeToString(b[:])
}
