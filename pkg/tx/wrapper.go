package tx

import "context"

type Wrapper interface {
	Wrap(ctx context.Context, fn func(context.Context) error) error
}
