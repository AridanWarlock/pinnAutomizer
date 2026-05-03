package tasksPlot

import (
	"context"
	"io"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/google/uuid"
)

type Input struct {
	TaskID uuid.UUID `validate:"required,uuid"`
	Writer io.Writer `validate:"required"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Usecase interface {
	DownloadTaskPlot(ctx context.Context, in Input) error
}
