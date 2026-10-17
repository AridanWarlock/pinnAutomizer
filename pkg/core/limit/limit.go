package limit

import (
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

// Limit is a positive SQL batch size. Its zero value is not usable.
type Limit struct {
	value int
}

func New(value int) (Limit, error) {
	l := Limit{value: value}
	return l, l.Validate()
}

// MustNew is for fixed application constants, not user input.
func MustNew(value int) Limit {
	l, err := New(value)
	if err != nil {
		panic(err)
	}
	return l
}

func (l Limit) Validate() error {
	if l.value <= 0 {
		return fmt.Errorf("%w: SQL limit must be positive", errs.ErrInvalidArgument)
	}
	return nil
}

func (l Limit) Int() int {
	return l.value
}

func (l Limit) Uint64() uint64 {
	return uint64(l.value)
}
