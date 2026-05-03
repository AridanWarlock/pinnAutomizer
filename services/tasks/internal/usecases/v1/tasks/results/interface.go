package tasksResults

import (
	"context"
	"io"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
	"github.com/google/uuid"
)

type DownloadType string

const (
	DownloadTypeOutput DownloadType = "output"
	DownloadTypeData   DownloadType = "data"
)

type Input struct {
	TaskID       uuid.UUID    `validate:"required,uuid"`
	Writer       io.Writer    `validate:"required"`
	DownloadType DownloadType `validate:"required,oneof=output data"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Usecase interface {
	DownloadTaskResults(ctx context.Context, in Input) error
}
