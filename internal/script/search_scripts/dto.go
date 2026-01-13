package search_scripts

import (
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/internal/domain/pagination"
	"pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type Input struct {
	userID uuid.UUID `validate:"required,uuid"`
	f      *domain.ScriptFilter
	p      pagination.Options `validate:"required"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	Scripts []*domain.Script
	Count   int
}
