package httpRequest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jsonDecoder"
	"github.com/go-playground/validator/v10"
)

type validatable interface {
	Validate() error
}

var requestValidator = validator.New(validator.WithRequiredStructEnabled())

func DecodeAndValidateRequest(w http.ResponseWriter, r *http.Request, dst any) error {
	err := jsonDecoder.ParseRequestBody(w, r, dst)
	switch {
	case err == nil:
	case errors.Is(err, jsonDecoder.ErrBadJson):
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			errs.ErrInvalidArgument,
		)
	case errors.Is(err, jsonDecoder.ErrEntityToLarge):
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			errs.ErrEntityToLarge,
		)
	default:
		return fmt.Errorf("unexpected decode json error: %w", err)
	}

	if err := requestValidator.Struct(dst); err != nil {
		return fmt.Errorf(
			"request validation: %v: %w",
			err,
			errs.ErrInvalidArgument,
		)
	}

	if v, ok := dst.(validatable); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf(
				"request validation: %v: %w",
				err,
				errs.ErrInvalidArgument,
			)
		}
	}

	return nil
}
