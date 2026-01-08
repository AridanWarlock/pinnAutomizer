package search_scripts

import (
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/internal/domain/pagination"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Input struct {
	userID uuid.UUID `validate:"required,uuid"`
	f      *domain.ScriptFilter
	p      pagination.Options `validate:"required"`
}

func (i Input) Validate(validate *validator.Validate) error {
	return validate.Struct(i)
}

type Output struct {
	Scripts []*domain.Script
	Count   int
}
