package database_test

import (
	"testing"
	"time"

	"github.com/go-jimu/template/internal/pkg/database"
)

func TestTimestamp(t *testing.T) {
	now := time.Now()
	ts := database.NewTimestamp(now)
	t.Log(ts)
	t.Log(now.UnixMicro())
}

func TestEmptyTimestamp(t *testing.T) {
	ts := database.NewTimestamp(time.Time{})
	t.Log(ts.Time.String())

	ts2 := database.NewNullTimestamp(database.UnixEpoch)
	t.Log(ts2.Time.String())
}
