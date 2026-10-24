package poolx

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func integrationPools(t *testing.T) (Pool, Pool) {
	t.Helper()
	dsn := os.Getenv("PINN_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set PINN_TEST_POSTGRES_DSN to a disposable PostgreSQL cluster")
	}
	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	admin, err := pgx.ConnectConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		assert.NoError(t, admin.Close(ctx))
	})
	name := "pinn_poolx_" + uuid.NewString()
	quoted := pgx.Identifier{name}.Sanitize()
	_, err = admin.Exec(t.Context(), "CREATE DATABASE "+quoted)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := admin.Exec(ctx, "DROP DATABASE "+quoted)
		assert.NoError(t, err)
	})
	config := Config{
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     int(cfg.Port),
		DB:       name,
		SslMode:  "disable",
		Timeout:  5 * time.Second,
	}
	first, err := New(config)
	require.NoError(t, err)
	t.Cleanup(first.Close)
	second, err := New(config)
	require.NoError(t, err)
	t.Cleanup(second.Close)
	_, err = first.Execx(t.Context(), sq.Expr("CREATE TABLE ownership (id text PRIMARY KEY)"))
	require.NoError(t, err)
	return first, second
}

func insertOwnership(ctx context.Context, client Pool, id string) error {
	_, err := client.Execx(ctx, sq.Expr("INSERT INTO ownership(id) VALUES ($1)", id))
	return err
}

func ownershipIDs(t *testing.T, client Pool) []string {
	t.Helper()
	var ids []string
	require.NoError(t, client.Selectx(t.Context(), &ids, sq.Expr("SELECT id FROM ownership ORDER BY id")))
	return ids
}

func TestClientDoesNotBorrowAnotherPoolTransactionIntegration(t *testing.T) {
	first, second := integrationPools(t)
	failure := errors.New("rollback first")
	err := first.InTransaction(t.Context(), func(ctx context.Context) error {
		require.NoError(t, insertOwnership(ctx, first, "first-rollback"))
		require.NoError(t, insertOwnership(ctx, second, "second-autocommit"))
		return failure
	})
	require.ErrorIs(t, err, failure)
	require.Equal(t, []string{"second-autocommit"}, ownershipIDs(t, first))
}

func TestNestedClientsCommitAndRollbackIndependentlyIntegration(t *testing.T) {
	first, second := integrationPools(t)
	failure := errors.New("rollback first")
	err := first.InTransaction(t.Context(), func(ctx context.Context) error {
		require.NoError(t, insertOwnership(ctx, first, "first-before"))
		require.NoError(t, second.InTransaction(ctx, func(ctx context.Context) error {
			require.NoError(t, insertOwnership(ctx, first, "first-inside-second"))
			require.NoError(t, insertOwnership(ctx, second, "second-commit"))
			err := first.InTransaction(ctx, func(context.Context) error { return nil })
			require.ErrorIs(t, err, errs.ErrInvalidArgument)
			return nil
		}))
		return failure
	})
	require.ErrorIs(t, err, failure)
	require.Equal(t, []string{"second-commit"}, ownershipIDs(t, first))
}

func TestCancellationRollsBackAndPoolRemainsUsableIntegration(t *testing.T) {
	first, _ := integrationPools(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	err := first.InTransaction(ctx, func(ctx context.Context) error {
		require.NoError(t, insertOwnership(ctx, first, "cancelled"))
		cancel()
		return ctx.Err()
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, ownershipIDs(t, first))
	require.NoError(t, first.InTransaction(t.Context(), func(ctx context.Context) error {
		return insertOwnership(ctx, first, "cancelled")
	}))
	require.Equal(t, []string{"cancelled"}, ownershipIDs(t, first))
}
