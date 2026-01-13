package domain

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID uuid.UUID

	Topic string
	Data  []byte

	CreatedAt time.Time
}
