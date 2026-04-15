package domain

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"

	"github.com/google/uuid"
)

type EquationType = string

const (
	EquationTypeHeat EquationType = "heat"
	EquationTypeWave EquationType = "wave"
)

type Equation struct {
	ID   uuid.UUID `validate:"required,uuid" json:"id"`
	Type string    `validate:"required,oneof=heat wave" json:"type"`
}

func NewEquation(equationType string) (Equation, error) {
	e := Equation{
		ID:   uuid.New(),
		Type: equationType,
	}

	if err := e.Validate(); err != nil {
		return Equation{}, err
	}
	return e, nil
}

func (e Equation) Validate() error {
	return validate.V.Struct(e)
}
