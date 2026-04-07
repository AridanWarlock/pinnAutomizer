package http_request

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/json_decoder"
	"github.com/go-playground/validator/v10"
)

type validatable interface {
	Validate() error
}

var requestValidator = validator.New()

func DecodeAndValidateRequest(w http.ResponseWriter, r *http.Request, dst any) error {
	err := json_decoder.ParseRequestBody(w, r, dst)
	switch {
	case err == nil:
	case errors.Is(err, json_decoder.ErrBadJson):
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			errs.ErrInvalidArgument,
		)
	case errors.Is(err, json_decoder.ErrEntityToLarge):
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			errs.ErrEntityToLarge,
		)
	default:
		panic(fmt.Sprintf("unexpected decode json: %v", err))
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
