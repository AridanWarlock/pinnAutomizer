package http_request

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/internal/errors"
	"github.com/AridanWarlock/pinnAutomizer/pkg/json_decoder"
	"github.com/go-playground/validator/v10"
)

var requestValidator = validator.New()

func DecodeAndValidateRequest(w http.ResponseWriter, r *http.Request, dst any) error {
	err := json_decoder.ParseRequestBody(w, r, dst)
	switch {
	case err == nil:
	case errors.Is(err, json_decoder.ErrBadJson):
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			core_errors.ErrInvalidArgument,
		)
	case errors.Is(err, json_decoder.ErrEntityToLarge):
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			core_errors.ErrEntityToLarge,
		)
	default:
		panic(fmt.Sprintf("unexpected decode json: %v", err))
	}

	if err := requestValidator.Struct(dst); err != nil {
		return fmt.Errorf(
			"request validation: %v: %w",
			err,
			core_errors.ErrInvalidArgument,
		)
	}

	return nil
}
