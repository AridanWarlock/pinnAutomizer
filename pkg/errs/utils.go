package errs

import (
	"context"
	"errors"
)

func OneOf(err error, errs ...error) bool {
	for _, e := range errs {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

func IsContextErr(err error) bool {
	return OneOf(err, context.Canceled, context.DeadlineExceeded)
}
