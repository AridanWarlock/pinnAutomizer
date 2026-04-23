package pagination

import (
	"errors"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

var (
	ErrInvalidPagination = errors.New("invalid pagination")
)

type Option func(opts *Options)

type Options struct {
	limit  *int        `validate:"min=1,max=100"`
	offset *int        `validate:"min=0"`
	sort   []SortField `validate:"dive,required"`
}

func NewOptions(opts ...Option) (Options, error) {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}

	if err := o.Validate(); err != nil {
		return Options{}, err
	}

	return o, nil
}

func (p *Options) Limit() *int {
	return p.limit
}

func (p *Options) Offset() *int {
	return p.offset
}

func (p *Options) OrderBy() []SortField {
	return p.sort
}

func (p *Options) Validate() error {
	if err := validate.V.Struct(p); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrInvalidPagination,
			err,
		)
	}
	return nil
}

type SortField struct {
	Name      string `validate:"required"`
	Direction string `validate:"required,oneof=ASC DESC"`
}

func (s SortField) String() string {
	return fmt.Sprintf("%s %s", s.Name, s.Direction)
}

func OrderBy(name string, direction string) SortField {
	return SortField{Name: name, Direction: direction}
}

func WithLimit(limit *int) Option {
	return func(opts *Options) { opts.limit = limit }
}

func WithOffset(offset *int) Option {
	return func(opts *Options) { opts.offset = offset }
}

func WithSortFields(fields ...SortField) Option {
	return func(opts *Options) { opts.sort = fields }
}
