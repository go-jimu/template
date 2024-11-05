package database

import (
	"testing"
	"time"
)

func TestTimestamp(t *testing.T) {
	now := time.Now()
	ts := NewTimestamp(now)
	t.Log(ts)
	t.Log(now.UnixMicro())
}
