package domain

import "github.com/google/uuid"

var zeroJti = Jti(uuid.Nil)

type Jti uuid.UUID

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
