package httpin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core/pagination"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func ParsePaginationOptions(r *http.Request) (*pagination.Options, error) {
	limit, err := QueryInt(r, "limit")
	if err != nil {
		return nil, err
	}
	offset, err := QueryInt(r, "offset")
	if err != nil {
		return nil, err
	}
	field := r.URL.Query().Get("sort")

	orderStr := strings.ToUpper(r.URL.Query().Get("order"))
	dir := "DESC"
	switch orderStr {
	case "DESC", "":
	case "ASC":
		dir = "ASC"
	default:
		return nil, fmt.Errorf(
			"invalid order: expected=[ASC, DESC] actual=%s",
			orderStr,
		)
	}

	opt := []pagination.Option{
		pagination.WithLimit(limit),
		pagination.WithOffset(offset),

		pagination.WithSortFields(pagination.SortField{
			Name:      field,
			Direction: dir,
		}),
	}

	opts, err := pagination.NewOptions(opt...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}
	return opts, nil
}
