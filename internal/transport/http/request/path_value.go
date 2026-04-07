package httpRequest

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
)

func GetIntPathValue(r *http.Request, key string) (int, error) {
	pathValue := r.PathValue(key)
	if pathValue == "" {
		return 0, fmt.Errorf(
			"path value by key %s is unset: %w",
			key,
			errs.ErrInvalidArgument,
		)
	}

	val, err := strconv.Atoi(pathValue)
	if err != nil {
		return 0, fmt.Errorf(
			"path value=%s by key=%s is not valid integer: %v: %w",
			pathValue,
			key,
			err,
			errs.ErrInvalidArgument,
		)
	}

	return val, nil
}
