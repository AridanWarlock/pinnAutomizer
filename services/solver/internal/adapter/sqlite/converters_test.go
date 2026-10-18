package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

func TestAttemptRowRoundTripPreservesDomainState(t *testing.T) {
	a := attempt(t)
	_, err := a.RecordContainer("container")
	require.NoError(t, err)
	_, err = a.RequestStart(time.Now(), time.Minute)
	require.NoError(t, err)
	code := 0
	_, err = a.Finish(&code, nil, time.Now())
	require.NoError(t, err)
	require.Equal(t, a, newAttemptRow(a).toDomain())
}

func TestAttemptRowPanicsOnInvalidPersistenceInvariant(t *testing.T) {
	for _, tc := range []struct {
		name    string
		corrupt func(*attemptRow)
	}{
		{"unknown state", func(row *attemptRow) {
			row.State = "unknown"
		}},
		{"missing deadline", func(row *attemptRow) {
			row.State = domain.AttemptStartRequested
			row.ContainerID = "container"
		}},
		{"missing run ID", func(row *attemptRow) {
			row.RunID = uuid.Nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := newAttemptRow(attempt(t))
			tc.corrupt(&row)
			require.Panics(t, func() {
				_ = row.toDomain()
			})
		})
	}
}

func TestEventRowPanicsOnCorruptPayloadOrIdentity(t *testing.T) {
	for _, tc := range []struct {
		name    string
		corrupt func(*eventRow)
	}{
		{"malformed headers", func(row *eventRow) {
			row.Headers = "{"
		}},
		{"empty topic", func(row *eventRow) {
			row.Topic = ""
		}},
		{"mismatched event ID", func(row *eventRow) {
			row.ID = uuid.New()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := newEventRow(event(t, attempt(t)))
			tc.corrupt(&row)
			require.Panics(t, func() {
				_ = row.toDomain()
			})
		})
	}
}

func TestPersistencePanicRollsBackRegistrationTransaction(t *testing.T) {
	repo := open(t, config(t))
	a := attempt(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _, err := repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	// SQL CHECKs do not prohibit a container ID in the accepted state.
	_, err = sqliteClient(repo).Database().Exec("UPDATE attempts SET container_id='unexpected' WHERE run_id=?", a.RunID)
	require.NoError(t, err)
	require.Panics(t, func() {
		_, _, _ = repo.RegisterAttempt(ctx, a)
	})
	_, err = sqliteClient(repo).Database().Exec("UPDATE attempts SET container_id='' WHERE run_id=?", a.RunID)
	require.NoError(t, err)
	stored, created, err := repo.RegisterAttempt(ctx, a)
	require.NoError(t, err)
	require.False(t, bool(created))
	require.True(t, stored.Matches(a))
}
