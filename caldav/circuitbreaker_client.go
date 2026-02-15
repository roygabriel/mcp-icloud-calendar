package caldav

import (
	"context"
	"time"
)

// CircuitBreakerClient wraps a CalendarService with circuit breaker protection.
type CircuitBreakerClient struct {
	inner   CalendarService
	breaker *CircuitBreaker
}

var _ CalendarService = (*CircuitBreakerClient)(nil)

// NewCircuitBreakerClient wraps the given client with circuit breaker logic.
func NewCircuitBreakerClient(inner CalendarService, breaker *CircuitBreaker) *CircuitBreakerClient {
	return &CircuitBreakerClient{
		inner:   inner,
		breaker: breaker,
	}
}

func (c *CircuitBreakerClient) execute(fn func() error) error {
	if err := c.breaker.Allow(); err != nil {
		return err
	}
	err := fn()
	if isServerError(err) {
		c.breaker.RecordFailure()
	} else {
		c.breaker.RecordSuccess()
	}
	return err
}

// DiscoverCalendarHomeSet discovers the calendar home set with circuit breaker protection.
func (c *CircuitBreakerClient) DiscoverCalendarHomeSet(ctx context.Context) (string, error) {
	var result string
	err := c.execute(func() error {
		var e error
		result, e = c.inner.DiscoverCalendarHomeSet(ctx)
		return e
	})
	return result, err
}

// ListCalendars lists calendars with circuit breaker protection.
func (c *CircuitBreakerClient) ListCalendars(ctx context.Context) ([]Calendar, error) {
	var result []Calendar
	err := c.execute(func() error {
		var e error
		result, e = c.inner.ListCalendars(ctx)
		return e
	})
	return result, err
}

// SearchEvents searches events with circuit breaker protection.
func (c *CircuitBreakerClient) SearchEvents(ctx context.Context, calendarPath string, startTime, endTime *time.Time) ([]Event, error) {
	var result []Event
	err := c.execute(func() error {
		var e error
		result, e = c.inner.SearchEvents(ctx, calendarPath, startTime, endTime)
		return e
	})
	return result, err
}

// CreateEvent creates an event with circuit breaker protection.
func (c *CircuitBreakerClient) CreateEvent(ctx context.Context, calendarPath string, event *Event) (string, error) {
	var result string
	err := c.execute(func() error {
		var e error
		result, e = c.inner.CreateEvent(ctx, calendarPath, event)
		return e
	})
	return result, err
}

// UpdateEvent updates an event with circuit breaker protection.
func (c *CircuitBreakerClient) UpdateEvent(ctx context.Context, eventPath string, update *EventUpdate) error {
	return c.execute(func() error {
		return c.inner.UpdateEvent(ctx, eventPath, update)
	})
}

// DeleteEvent deletes an event with circuit breaker protection.
func (c *CircuitBreakerClient) DeleteEvent(ctx context.Context, eventPath string) error {
	return c.execute(func() error {
		return c.inner.DeleteEvent(ctx, eventPath)
	})
}

// GetEventPath delegates to the inner client (pure computation, no breaker needed).
func (c *CircuitBreakerClient) GetEventPath(calendarPath, eventID string) string {
	return c.inner.GetEventPath(calendarPath, eventID)
}
