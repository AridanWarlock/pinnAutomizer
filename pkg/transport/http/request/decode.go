package request

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type validatable interface {
	Validate() error
}

func DecodeAndValidate(w http.ResponseWriter, r *http.Request, dst any) error {
	err := decodeRequestBody(w, r, dst)
	switch {
	case err == nil:
	case errors.Is(err, ErrBadJson):
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			errs.ErrInvalidArgument,
		)
	case errors.Is(err, ErrEntityToLarge):
		return fmt.Errorf("decode json: %w: ", err)
	default:
		return fmt.Errorf("unexpected decode json error: %w", err)
	}

	if v, ok := dst.(validatable); ok {
		err = v.Validate()
	} else {
		err = validate.V.Struct(dst)
	}

	if err != nil {
		return fmt.Errorf(
			"request validation: %w: %v",
			errs.ErrInvalidArgument,
			err,
		)
	}

	return nil
}
