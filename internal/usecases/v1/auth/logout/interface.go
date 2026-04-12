package authLogout

import (
	"context"
)

type Usecase interface {
	Logout(ctx context.Context) error
}
