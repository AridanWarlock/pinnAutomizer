package domain

import (
	"fmt"
	"pinnAutomizer/pkg/validate"
	"time"

	"github.com/google/uuid"
)

type Script struct {
	ID         uuid.UUID
	Filename   string
	Path       string
	UploadTime time.Time
	Text       string
	UserID     uuid.UUID
}

func NewScript(filename string, path string, userID uuid.UUID) (*Script, error) {
	id := uuid.New()

	s := &Script{
		ID:         id,
		Filename:   filename,
		Path:       path,
		UploadTime: time.Now(),
		Text:       "",
		UserID:     userID,
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Script) Validate() error {
	err := validate.V.Struct(s)
	if err != nil {
		return fmt.Errorf("script.Validate: %w", err)
	}

	return nil
}

type ScriptFilter struct {
	IDs            []uuid.UUID `validate:"dive,required,uuid"`
	Filename       *string
	UploadTimeFrom *time.Time
	UploadTimeTo   *time.Time `validate:"gtfield=UploadTimeFrom"`
}
