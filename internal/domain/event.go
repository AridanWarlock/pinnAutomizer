package domain

import (
	"github.com/google/uuid"
	"time"
)

type Event struct {
	ID uuid.UUID

	Topic string
	Data  []byte

	CreatedAt time.Time
}
