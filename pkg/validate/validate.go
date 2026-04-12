package validate

import "github.com/go-playground/validator/v10"

var V = validator.New(validator.WithRequiredStructEnabled())

func Caller(s any) func() error {
	return func() error {
		return V.Struct(s)
	}
}
