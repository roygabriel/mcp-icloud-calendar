package caldav

import (
	"context"
	"log/slog"
	"math"
	"time"
)

// RetryClient wraps a CalendarService with retry logic using exponential backoff.
// Only idempotent operations are retried.
type RetryClient struct {
	inner     CalendarService
	maxRetry  int
	baseDelay time.Duration
}

var _ CalendarService = (*RetryClient)(nil)

// NewRetryClient wraps the given client with retry logic.
func NewRetryClient(inner CalendarService, maxRetries int, baseDelay time.Duration) *RetryClient {
	return &RetryClient{
		inner:     inner,
		maxRetry:  maxRetries,
		baseDelay: baseDelay,
	}
}

func (r *RetryClient) retry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetry; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt == r.maxRetry {
			break
		}
		delay := r.baseDelay * time.Duration(math.Pow(2, float64(attempt)))
		slog.Warn("retrying operation", "operation", operation, "attempt", attempt+1, "delay", delay, "error", lastErr)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

// DiscoverCalendarHomeSet retries (idempotent).
func (r *RetryClient) DiscoverCalendarHomeSet(ctx context.Context) (string, error) {
	var result string
	err := r.retry(ctx, "DiscoverCalendarHomeSet", func() error {
		var e error
		result, e = r.inner.DiscoverCalendarHomeSet(ctx)
		return e
	})
	return result, err
}

// ListCalendars retries (idempotent).
func (r *RetryClient) ListCalendars(ctx context.Context) ([]Calendar, error) {
	var result []Calendar
	err := r.retry(ctx, "ListCalendars", func() error {
		var e error
		result, e = r.inner.ListCalendars(ctx)
		return e
	})
	return result, err
}

// SearchEvents retries (idempotent).
func (r *RetryClient) SearchEvents(ctx context.Context, calendarPath string, startTime, endTime *time.Time) ([]Event, error) {
	var result []Event
	err := r.retry(ctx, "SearchEvents", func() error {
		var e error
		result, e = r.inner.SearchEvents(ctx, calendarPath, startTime, endTime)
		return e
	})
	return result, err
}

// CreateEvent does NOT retry (not idempotent).
func (r *RetryClient) CreateEvent(ctx context.Context, calendarPath string, event *Event) (string, error) {
	return r.inner.CreateEvent(ctx, calendarPath, event)
}

// UpdateEvent does NOT retry (not idempotent).
func (r *RetryClient) UpdateEvent(ctx context.Context, eventPath string, update *EventUpdate) error {
	return r.inner.UpdateEvent(ctx, eventPath, update)
}

// DeleteEvent retries (idempotent).
func (r *RetryClient) DeleteEvent(ctx context.Context, eventPath string) error {
	return r.retry(ctx, "DeleteEvent", func() error {
		return r.inner.DeleteEvent(ctx, eventPath)
	})
}

// GetEventPath delegates to the inner client.
func (r *RetryClient) GetEventPath(calendarPath, eventID string) string {
	return r.inner.GetEventPath(calendarPath, eventID)
}
