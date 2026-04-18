package httpin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func QueryInt(r *http.Request, key string) (*int, error) {
	param := r.URL.Query().Get(key)
	if param == "" {
		return nil, nil
	}

	val, err := strconv.Atoi(param)
	if err != nil {
		return nil, fmt.Errorf(
			"param=%s by key=%s is not valid integer: %v: %w",
			param,
			key,
			err,
			errs.ErrInvalidArgument,
		)
	}

	return &val, err
}
