package httpin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/google/uuid"
)

func Path(r *http.Request, key string) (string, error) {
	value := r.PathValue(key)

	if value == "" {
		return "", fmt.Errorf(
			"path value by key %s is unset: %w",
			key,
			errs.ErrInvalidArgument,
		)
	}

	return value, nil
}

func PathInt(r *http.Request, key string) (int, error) {
	pathValue, err := Path(r, key)
	if err != nil {
		return 0, err
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

func PathUuid(r *http.Request, key string) (uuid.UUID, error) {
	pathValue, err := Path(r, key)
	if err != nil {
		return uuid.Nil, err
	}

	id, err := uuid.Parse(pathValue)
	if err != nil {
		return uuid.Nil, fmt.Errorf(
			"path value by key %s is not valid uuid: %w",
			key,
			errs.ErrInvalidArgument,
		)
	}

	return id, nil
}
