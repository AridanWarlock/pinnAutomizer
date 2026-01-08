package domain

import "github.com/google/uuid"

type ToTranslate struct {
	ID   uuid.UUID
	Path string
}

type FromTranslate struct {
	ScriptID uuid.UUID
	Text     string
}
