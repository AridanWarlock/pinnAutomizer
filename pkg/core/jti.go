package core

import "github.com/google/uuid"

var zeroJti = Jti(uuid.Nil)

type Jti uuid.UUID

func NewJti(id uuid.UUID) (Jti, error) {
	jti := Jti(id)
	if err := jti.Validate(); err != nil {
		return zeroJti, err
	}
	return jti, nil
}

func (i Jti) IsZero() bool {
	return i == zeroJti
}

func (i Jti) String() string {
	return uuid.UUID(i).String()
}

func (i Jti) ToRedisKey() string {
	return "sess:jti:" + i.String()
}

func (i Jti) Validate() error {
	if i.IsZero() {
		return ErrInvalidJti
	}
	return nil
}
