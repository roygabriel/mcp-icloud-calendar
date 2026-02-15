package logging

import (
	"context"
	"log/slog"
	"strings"
)

// RedactingHandler wraps an slog.Handler and replaces known secret strings
// with [REDACTED] in log messages and string attributes.
type RedactingHandler struct {
	inner   slog.Handler
	secrets []string
}

// NewRedactingHandler creates a handler that redacts the given secrets from all output.
// Empty strings in secrets are filtered out to avoid replacing everything.
func NewRedactingHandler(inner slog.Handler, secrets []string) *RedactingHandler {
	filtered := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return &RedactingHandler{inner: inner, secrets: filtered}
}

func (h *RedactingHandler) redact(s string) string {
	for _, secret := range h.secrets {
		s = strings.ReplaceAll(s, secret, "[REDACTED]")
	}
	return s
}

func (h *RedactingHandler) redactAttr(a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindString:
		a.Value = slog.StringValue(h.redact(a.Value.String()))
	case slog.KindGroup:
		attrs := a.Value.Group()
		redacted := make([]slog.Attr, len(attrs))
		for i, ga := range attrs {
			redacted[i] = h.redactAttr(ga)
		}
		a.Value = slog.GroupValue(redacted...)
	}
	return a
}

// Enabled reports whether the handler handles records at the given level.
func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle redacts secrets from the record message and attributes before delegating.
func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Message = h.redact(r.Message)
	var redacted []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		redacted = append(redacted, h.redactAttr(a))
		return true
	})
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	newRecord.AddAttrs(redacted...)
	return h.inner.Handle(ctx, newRecord)
}

// WithAttrs returns a new RedactingHandler with additional attributes.
func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		redacted[i] = h.redactAttr(a)
	}
	return &RedactingHandler{
		inner:   h.inner.WithAttrs(redacted),
		secrets: h.secrets,
	}
}

// WithGroup returns a new RedactingHandler with the given group name.
func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{
		inner:   h.inner.WithGroup(name),
		secrets: h.secrets,
	}
}
