package domain

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Script struct {
	ID         uuid.UUID `json:"id"`
	Filename   string    `json:"filename"`
	Path       string    `json:"path"`
	UploadTime time.Time `json:"uploadTime"`
	Text       string    `json:"text"`
	UserID     uuid.UUID `json:"userId"`
}

var scriptsValidator = validator.New(validator.WithRequiredStructEnabled())

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
	err := scriptsValidator.Struct(s)
	if err != nil {
		return fmt.Errorf("scriptsValidator.Sctuct: %w", err)
	}

	return nil
}

type ScriptFilter struct {
	IDs            []uuid.UUID `validate:"dive,required,uuid"`
	Filename       *string
	UploadTimeFrom *time.Time
	UploadTimeTo   *time.Time `validate:"gtfield=UploadTimeFrom"`
}
