package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func queryClient(t *testing.T) *SQLite {
	t.Helper()
	path := filepath.Join(t.TempDir(), "query.db")
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE records(id INTEGER PRIMARY KEY, full_name TEXT NOT NULL UNIQUE, note TEXT)")
	require.NoError(t, err)
	require.NoError(t, db.Close())
	client, err := New(Config{Path: path, BusyTimeout: time.Second, Timeout: time.Second})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})
	return client
}

type record struct {
	ID   int     `db:"id"`
	Name string  `db:"full_name"`
	Note *string `db:"note"`
}

func TestQueryBuildersScanStructsAndReportAffectedRows(t *testing.T) {
	client := queryClient(t)
	ctx := context.Background()
	tag, err := client.Execx(ctx, sq.Insert("records").Columns("id", "full_name", "note").Values(1, "first", nil).Values(2, "second", "note"))
	require.NoError(t, err)
	require.Equal(t, 2, tag.RowsAffected())
	var found record
	require.NoError(t, client.Getx(ctx, &found, sq.Select("full_name", "id", "note").From("records").Where(sq.Eq{"id": 1})))
	require.Equal(t, "first", found.Name)
	require.Nil(t, found.Note)
	var all []record
	require.NoError(t, client.Selectx(ctx, &all, sq.Select("id", "full_name", "note").From("records").OrderBy("id")))
	require.Len(t, all, 2)
	require.Equal(t, "note", *all[1].Note)
	tag, err = client.Execx(ctx, sq.Update("records").Set("full_name", "changed").Where(sq.Eq{"id": 1}))
	require.NoError(t, err)
	require.Equal(t, 1, tag.RowsAffected())
	tag, err = client.Execx(ctx, sq.Delete("records").Where(sq.Eq{"id": 99}))
	require.NoError(t, err)
	require.Zero(t, tag.RowsAffected())
	err = client.Getx(ctx, &found, sq.Select("id", "full_name", "note").From("records").Where(sq.Eq{"id": 99}))
	require.ErrorIs(t, err, errs.ErrNotFound)
	_, err = client.Execx(ctx, sq.Insert("records").Columns("id", "full_name").Values(1, "duplicate"))
	require.ErrorIs(t, err, errs.ErrConflict)
	var empty []record
	require.NoError(t, client.Selectx(ctx, &empty, sq.Select("id", "full_name", "note").From("records").Where(sq.Eq{"id": 99})))
	require.Empty(t, empty)
	require.Error(t, client.Getx(ctx, &found, sq.Select("full_name AS unknown_column").From("records").Where(sq.Eq{"id": 1})))
	_, err = client.Execx(ctx, sq.Insert("records"))
	require.ErrorContains(t, err, "to sql")
}

func TestTransactionCommitRollbackPanicAndCancellation(t *testing.T) {
	client := queryClient(t)
	ctx := context.Background()
	insert := func(ctx context.Context, id int) error {
		_, err := client.Execx(ctx, sq.Insert("records").Columns("id", "full_name").Values(id, "row"))
		return err
	}
	require.NoError(t, client.InTransaction(ctx, func(ctx context.Context) error {
		return insert(ctx, 1)
	}))
	_, err := client.Execx(ctx, sq.Delete("records"))
	require.NoError(t, err)
	failure := errors.New("later effect failed")
	err = client.InTransaction(ctx, func(ctx context.Context) error {
		require.NoError(t, insert(ctx, 2))
		return failure
	})
	require.ErrorIs(t, err, failure)
	require.Panics(t, func() {
		_ = client.InTransaction(ctx, func(ctx context.Context) error {
			require.NoError(t, insert(ctx, 3))
			panic("abort")
		})
	})
	canceled, cancel := context.WithCancel(ctx)
	err = client.InTransaction(canceled, func(ctx context.Context) error {
		require.NoError(t, insert(ctx, 4))
		cancel()
		return ctx.Err()
	})
	require.ErrorIs(t, err, context.Canceled)
	var count int
	require.NoError(t, client.Getx(ctx, &count, sq.Select("COUNT(*)").From("records")))
	require.Zero(t, count)
	require.NoError(t, client.InTransaction(ctx, func(ctx context.Context) error {
		return insert(ctx, 5)
	}))
	require.ErrorIs(t, client.InTransaction(ctx, func(ctx context.Context) error {
		return client.InTransaction(ctx, func(context.Context) error {
			return nil
		})
	}), errs.ErrInvalidArgument)
}

func TestTransactionContextDoesNotLeakToAnotherClient(t *testing.T) {
	first := queryClient(t)
	second := queryClient(t)
	ctx := context.Background()
	failure := errors.New("rollback first")
	err := first.InTransaction(ctx, func(ctx context.Context) error {
		_, err := second.Execx(ctx, sq.Insert("records").Columns("id", "full_name").Values(1, "second database"))
		require.NoError(t, err)
		return failure
	})
	require.ErrorIs(t, err, failure)
	var count int
	require.NoError(t, first.Getx(ctx, &count, sq.Select("COUNT(*)").From("records")))
	require.Zero(t, count)
	require.NoError(t, second.Getx(ctx, &count, sq.Select("COUNT(*)").From("records")))
	require.Equal(t, 1, count)
}

func TestTransactionsKeepBindingsAcrossClients(t *testing.T) {
	first := queryClient(t)
	second := queryClient(t)
	failure := errors.New("rollback first only")
	err := first.InTransaction(context.Background(), func(ctx context.Context) error {
		if err := second.InTransaction(ctx, func(ctx context.Context) error {
			if _, err := first.Execx(ctx, sq.Insert("records").Columns("id", "full_name").Values(1, "first")); err != nil {
				return err
			}
			_, err := second.Execx(ctx, sq.Insert("records").Columns("id", "full_name").Values(1, "second"))
			return err
		}); err != nil {
			return err
		}
		return failure
	})
	require.ErrorIs(t, err, failure)
	var count int
	require.NoError(t, first.Getx(context.Background(), &count, sq.Select("COUNT(*)").From("records")))
	require.Zero(t, count)
	require.NoError(t, second.Getx(context.Background(), &count, sq.Select("COUNT(*)").From("records")))
	require.Equal(t, 1, count)
}

func TestNestedTransactionIsRejectedAcrossAnotherClient(t *testing.T) {
	first := queryClient(t)
	second := queryClient(t)
	err := first.InTransaction(context.Background(), func(ctx context.Context) error {
		return second.InTransaction(ctx, func(ctx context.Context) error {
			return first.InTransaction(ctx, func(context.Context) error {
				t.Error("nested transaction callback must not run")
				return nil
			})
		})
	})
	require.ErrorIs(t, err, errs.ErrInvalidArgument)
}
