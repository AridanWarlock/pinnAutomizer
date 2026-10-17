package sqlite

import (
	"context"
	"errors"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/require"

	shared "github.com/AridanWarlock/pinnAutomizer/pkg/sqlite"
)

type affectedRows int

func (n affectedRows) RowsAffected() int {
	return int(n)
}

func TestRecordContainerDoesNotReportChangedWhenCommitFails(t *testing.T) {
	failure := errors.New("commit failed")
	a := attempt(t)
	client := &MockSQLite{
		GetxFunc: func(_ context.Context, dst any, _ shared.Sqlizer) error {
			*dst.(*attemptRow) = fromAttempt(a)
			return nil
		},
		ExecxFunc: func(context.Context, shared.Sqlizer) (shared.Tag, error) {
			return affectedRows(1), nil
		},
		InTransactionFunc: func(ctx context.Context, fn func(context.Context) error) error {
			if err := fn(ctx); err != nil {
				return err
			}
			return failure
		},
	}
	repo := &Repository{sqlite: client, sb: sq.StatementBuilder.PlaceholderFormat(sq.Question)}
	changed, err := repo.RecordContainer(context.Background(), a.RunID, "container")
	require.ErrorIs(t, err, failure)
	require.False(t, bool(changed))
}
