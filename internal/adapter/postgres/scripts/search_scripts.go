package scripts

import (
	"context"
	"fmt"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/internal/domain/pagination"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func (r *ScriptsRepository) SearchScripts(
	ctx context.Context,
	userID uuid.UUID,
	f *domain.ScriptFilter,
	p pagination.Options,
) ([]*domain.Script, error) {
	if f == nil {
		f = &domain.ScriptFilter{}
	}

	q := r.sb.
		Select(scriptsTableColumns...).
		From(scriptsTable).
		Where(sq.Eq{scriptsTableColumnUserID: userID})

	if len(f.IDs) > 0 {
		q = q.Where(sq.Eq{scriptsTableColumnID: f.IDs})
	}
	if v := f.Filename; v != nil && strings.TrimSpace(*v) != "" {
		q = q.Where(sq.ILike{scriptsTableColumnFilename: "%" + *v + "%"})
	}
	if v := f.UploadTimeFrom; v != nil {
		q = q.Where(sq.GtOrEq{scriptsTableColumnUploadTime: *v})
	}
	if v := f.UploadTimeTo; v != nil {
		q = q.Where(sq.LtOrEq{scriptsTableColumnUploadTime: *v})
	}

	orderSql := buildOrderBy(p.OrderBy())
	if len(orderSql) == 0 {
		orderSql = []string{scriptsTableColumnUploadTime + " DESC"}
	}

	q = q.OrderBy(orderSql...)

	if l := p.Limit(); l > 0 {
		q = q.Limit(uint64(l))
	}
	if o := p.Offset(); o > 0 {
		q = q.Offset(uint64(o))
	}

	var rows []ScriptRow
	if err := r.pool.Selectx(ctx, &rows, q); err != nil {
		return nil, err
	}

	out := make([]*domain.Script, 0, len(rows))
	for i := range rows {
		out = append(out, ToModel(&rows[i]))
	}
	return out, nil
}

func buildOrderBy(fields []pagination.SortField) []string {
	if len(fields) == 0 {
		return nil
	}

	whiteList := map[string]bool{
		scriptsTableColumnUploadTime: true,
		scriptsTableColumnFilename:   true,
	}

	var out []string
	for _, f := range fields {
		name := strings.ToLower(strings.TrimSpace(f.Name))
		if !whiteList[name] {
			continue
		}

		dir := "ASC"
		if f.Desc {
			dir = "DESC"
		}

		out = append(out, fmt.Sprintf("%s %s", name, dir))
	}
	return out
}
