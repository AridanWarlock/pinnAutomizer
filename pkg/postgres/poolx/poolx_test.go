package poolx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func TestInitializationOwnsPoolUntilSuccessfulPing(t *testing.T) {
	t.Parallel()
	for _, fails := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "ping failure"}[fails], func(t *testing.T) {
			failure := errors.New("ping failed")
			postgres := &MockPostgres{
				PingFunc: func(context.Context) error {
					if fails {
						return failure
					}
					return nil
				},
				CloseFunc: func() {},
			}
			client, err := newPoolx(t.Context(), postgres, time.Second)
			if fails {
				require.ErrorIs(t, err, failure)
				require.Nil(t, client)
				require.Len(t, postgres.CloseCalls(), 1)
			} else {
				require.NoError(t, err)
				require.Empty(t, postgres.CloseCalls(), "successful constructor transfers ownership")
				client.Close()
				require.Len(t, postgres.CloseCalls(), 1)
			}
		})
	}
}

type testTransaction struct {
	pgx.Tx
	committed     bool
	rolledBack    bool
	rollbackError error
	commitError   error
}

func (tx *testTransaction) Commit(context.Context) error {
	if tx.commitError != nil {
		return tx.commitError
	}
	tx.committed = true
	return nil
}

func (tx *testTransaction) Rollback(context.Context) error {
	if tx.committed {
		return pgx.ErrTxClosed
	}
	tx.rolledBack = true
	return tx.rollbackError
}

func TestTransactionsKeepTheirOwnerThroughOtherClients(t *testing.T) {
	t.Parallel()
	firstTx, secondTx := &testTransaction{}, &testTransaction{}
	first := &Poolx{
		postgres: &MockPostgres{BeginFunc: func(context.Context) (pgx.Tx, error) { return firstTx, nil }},
		timeout:  time.Second,
	}
	second := &Poolx{
		postgres: &MockPostgres{BeginFunc: func(context.Context) (pgx.Tx, error) { return secondTx, nil }},
		timeout:  time.Second,
	}
	failure := errors.New("rollback first")
	err := first.InTransaction(t.Context(), func(ctx context.Context) error {
		require.Same(t, firstTx, first.executor(ctx))
		require.Same(t, second.postgres, second.executor(ctx), "another client must use its own pool")
		require.NoError(t, second.InTransaction(ctx, func(ctx context.Context) error {
			require.Same(t, firstTx, first.executor(ctx), "second context must not hide first ownership")
			require.Same(t, secondTx, second.executor(ctx))
			require.ErrorIs(t, first.InTransaction(ctx, func(context.Context) error {
				t.Fatal("nested transaction must not execute")
				return nil
			}), errs.ErrInvalidArgument)
			return nil
		}))
		return failure
	})
	require.ErrorIs(t, err, failure)
	require.True(t, firstTx.rolledBack)
	require.False(t, firstTx.committed)
	require.True(t, secondTx.committed)
	require.False(t, secondTx.rolledBack)
}

func TestTransactionRollsBackOnPanic(t *testing.T) {
	t.Parallel()
	tx := &testTransaction{}
	client := &Poolx{
		postgres: &MockPostgres{BeginFunc: func(context.Context) (pgx.Tx, error) { return tx, nil }},
		timeout:  time.Second,
	}
	require.PanicsWithValue(t, "failure", func() {
		_ = client.InTransaction(t.Context(), func(context.Context) error { panic("failure") })
	})
	require.True(t, tx.rolledBack)
	require.False(t, tx.committed)
}

func TestNilTransactionCallbackDoesNotBegin(t *testing.T) {
	t.Parallel()
	postgres := &MockPostgres{}
	client := &Poolx{postgres: postgres, timeout: time.Second}
	require.ErrorIs(t, client.InTransaction(t.Context(), nil), errs.ErrInvalidArgument)
	require.Empty(t, postgres.BeginCalls())
}

func TestRollbackFailurePreservesTheOriginalErrorWithoutLogger(t *testing.T) {
	t.Parallel()
	for _, commitFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "callback error", true: "commit error"}[commitFails], func(t *testing.T) {
			failure := errors.New("original error")
			rollbackFailure := errors.New("rollback error")
			tx := &testTransaction{rollbackError: rollbackFailure}
			if commitFails {
				tx.commitError = failure
			}
			client := &Poolx{
				postgres: &MockPostgres{BeginFunc: func(context.Context) (pgx.Tx, error) { return tx, nil }},
				timeout:  time.Second,
			}
			err := client.InTransaction(t.Context(), func(context.Context) error {
				if commitFails {
					return nil
				}
				return failure
			})
			require.ErrorIs(t, err, failure)
			require.ErrorIs(t, err, rollbackFailure)
			require.True(t, tx.rolledBack)
		})
	}
}
