package logging

import (
	"log/slog"
	"testing"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		name      string
		levelStr  string
		wantLevel slog.Level
	}{
		{"debug level", "DEBUG", slog.LevelDebug},
		{"info level", "INFO", slog.LevelInfo},
		{"warn level", "WARN", slog.LevelWarn},
		{"warning level", "WARNING", slog.LevelWarn},
		{"error level", "ERROR", slog.LevelError},
		{"default from empty", "", slog.LevelInfo},
		{"default from unknown", "TRACE", slog.LevelInfo},
		{"case insensitive", "debug", slog.LevelDebug},
		{"mixed case", "Warn", slog.LevelWarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := Setup(tt.levelStr)
			if logger == nil {
				t.Fatal("Setup returned nil logger")
			}
		})
	}
}
