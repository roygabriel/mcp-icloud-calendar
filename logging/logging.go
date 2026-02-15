// Package logging provides structured JSON logging with optional secret redaction.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes structured JSON logging on stderr with the given level.
func Setup(levelStr string) *slog.Logger {
	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// SetupWithRedaction initializes structured logging with secret redaction.
// Known secrets are replaced with [REDACTED] in log messages and attributes.
func SetupWithRedaction(levelStr string, secrets []string) *slog.Logger {
	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	jsonHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	handler := NewRedactingHandler(jsonHandler, secrets)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
