package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func newTestLogger(buf *bytes.Buffer, secrets []string) *slog.Logger {
	jsonHandler := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewRedactingHandler(jsonHandler, secrets)
	return slog.New(handler)
}

func TestRedactingHandler_MessageRedaction(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, []string{"supersecret"})

	logger.Info("token is supersecret here")

	output := buf.String()
	if strings.Contains(output, "supersecret") {
		t.Errorf("output contains secret: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Errorf("output missing [REDACTED]: %s", output)
	}
}

func TestRedactingHandler_AttributeRedaction(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, []string{"mysecrettoken"})

	logger.Info("request", "auth", "Bearer mysecrettoken")

	output := buf.String()
	if strings.Contains(output, "mysecrettoken") {
		t.Errorf("output contains secret in attribute: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Errorf("output missing [REDACTED]: %s", output)
	}
}

func TestRedactingHandler_GroupAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, []string{"password123"})

	logger.Info("grouped", slog.Group("credentials", slog.String("pass", "password123")))

	output := buf.String()
	if strings.Contains(output, "password123") {
		t.Errorf("output contains secret in group: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Errorf("output missing [REDACTED]: %s", output)
	}
}

func TestRedactingHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, []string{"secretval"})

	childLogger := logger.With("key", "secretval")
	childLogger.Info("test message")

	output := buf.String()
	if strings.Contains(output, "secretval") {
		t.Errorf("output contains secret via WithAttrs: %s", output)
	}
}

func TestRedactingHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, []string{"groupsecret"})

	childLogger := logger.WithGroup("mygroup")
	childLogger.Info("msg", "field", "groupsecret")

	output := buf.String()
	if strings.Contains(output, "groupsecret") {
		t.Errorf("output contains secret via WithGroup: %s", output)
	}
}

func TestRedactingHandler_EmptySecrets(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, []string{""})

	logger.Info("hello world")

	output := buf.String()
	if strings.Contains(output, "[REDACTED]") {
		t.Errorf("empty secret should not cause redaction: %s", output)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("message should be intact: %s", output)
	}
}

func TestRedactingHandler_MultipleSecrets(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, []string{"secret1", "secret2"})

	logger.Info("has secret1 and secret2")

	output := buf.String()
	if strings.Contains(output, "secret1") || strings.Contains(output, "secret2") {
		t.Errorf("output contains secrets: %s", output)
	}
	// Should have two [REDACTED] replacements.
	count := strings.Count(output, "[REDACTED]")
	if count != 2 {
		t.Errorf("expected 2 [REDACTED], got %d in: %s", count, output)
	}
}

func TestRedactingHandler_NoSecrets(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, nil)

	logger.Info("normal message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "normal message") {
		t.Errorf("message should pass through unchanged: %s", output)
	}
}

func TestRedactingHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	handler := NewRedactingHandler(jsonHandler, nil)

	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug should not be enabled when level is warn")
	}
	if !handler.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("warn should be enabled")
	}
}

func TestSetupWithRedaction(t *testing.T) {
	logger := SetupWithRedaction("DEBUG", []string{"token123"})
	if logger == nil {
		t.Fatal("SetupWithRedaction returned nil logger")
	}
}
