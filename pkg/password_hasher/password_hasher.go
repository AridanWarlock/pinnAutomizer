package password_hasher

import (
	"golang.org/x/crypto/bcrypt"
)

const hashingCost = 12

type Hasher struct {
}

func New() *Hasher {
	return &Hasher{}
}

func (h *Hasher) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), hashingCost)

	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (h *Hasher) CompareHashAndPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
