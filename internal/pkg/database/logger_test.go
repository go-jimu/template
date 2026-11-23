package database

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"xorm.io/xorm/log"
)

// Intent: xorm reports through Error the failures it swallows rather than
// returns, so they reach the log at Error under the default info level.
func TestXormSlogReportsErrorsAtErrorLevel(t *testing.T) {
	var logs bytes.Buffer
	logger := NewXormSlog(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo})))

	logger.Error("scan failed")
	logger.Errorf("insert failed: %v", "version")

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("logged %d lines, want 2: %q", len(lines), logs.String())
	}
	for _, line := range lines {
		if !strings.Contains(line, `"level":"ERROR"`) {
			t.Fatalf("xorm error logged as %q, want level ERROR", line)
		}
	}
}

// Intent: Level reports the most verbose level the handler enables.
func TestXormSlogLevelIsTheLowestEnabledLevel(t *testing.T) {
	for _, tc := range []struct {
		enabled slog.Level
		want    log.LogLevel
	}{
		{slog.LevelDebug, log.LOG_DEBUG},
		{slog.LevelInfo, log.LOG_INFO},
		{slog.LevelWarn, log.LOG_WARNING},
		{slog.LevelError, log.LOG_ERR},
	} {
		logger := NewXormSlog(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: tc.enabled})))
		if got := logger.Level(); got != tc.want {
			t.Errorf("Level() with %s enabled = %v, want %v", tc.enabled, got, tc.want)
		}
	}
}
