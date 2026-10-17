package sqlite

import (
	"context"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/require"
)

func TestTimestampKeepsNanosecondsAndUTC(t *testing.T) {
	original := time.Date(2026, 10, 17, 15, 20, 21, 123456789, time.FixedZone("offset", 3*3600))
	value, err := Timestamp(original).Value()
	require.NoError(t, err)
	var scanned SQLTime
	require.NoError(t, scanned.Scan(value))
	require.True(t, original.Equal(scanned.Time))
	require.Equal(t, time.UTC, scanned.Location())
	require.Nil(t, NullableTimestamp(nil))
	require.Nil(t, Time(nil))
	require.True(t, Time(NullableTimestamp(&original)).Equal(original))
}

func TestTimestampRejectsWrongStorageTypeAndOverflow(t *testing.T) {
	var scanned SQLTime
	require.Error(t, scanned.Scan("not an integer"))
	_, err := Timestamp(time.Date(2500, 1, 1, 0, 0, 0, 0, time.UTC)).Value()
	require.Error(t, err)
}

func TestTimestampScansNullableSQLColumns(t *testing.T) {
	client := queryClient(t)
	ctx := context.Background()
	_, err := client.Database().Exec("CREATE TABLE times(created_at INTEGER NOT NULL, deadline INTEGER)")
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = client.Execx(ctx, sq.Insert("times").Columns("created_at", "deadline").Values(Timestamp(now), NullableTimestamp(nil)))
	require.NoError(t, err)
	var row struct {
		CreatedAt SQLTime  `db:"created_at"`
		Deadline  *SQLTime `db:"deadline"`
	}
	require.NoError(t, client.Getx(ctx, &row, sq.Select("created_at", "deadline").From("times")))
	require.True(t, now.Equal(row.CreatedAt.Time))
	require.Nil(t, row.Deadline)
}
