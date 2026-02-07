package caldav

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitedClient wraps a CalendarService with token bucket rate limiting.
type RateLimitedClient struct {
	inner   CalendarService
	limiter *rate.Limiter
}

var _ CalendarService = (*RateLimitedClient)(nil)

// NewRateLimitedClient wraps the given client with rate limiting.
func NewRateLimitedClient(inner CalendarService, rps float64, burst int) *RateLimitedClient {
	return &RateLimitedClient{
		inner:   inner,
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

func (r *RateLimitedClient) wait(ctx context.Context) error {
	if err := r.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit exceeded: %w", err)
	}
	return nil
}

func (r *RateLimitedClient) DiscoverCalendarHomeSet(ctx context.Context) (string, error) {
	if err := r.wait(ctx); err != nil {
		return "", err
	}
	return r.inner.DiscoverCalendarHomeSet(ctx)
}

func (r *RateLimitedClient) ListCalendars(ctx context.Context) ([]Calendar, error) {
	if err := r.wait(ctx); err != nil {
		return nil, err
	}
	return r.inner.ListCalendars(ctx)
}

func (r *RateLimitedClient) SearchEvents(ctx context.Context, calendarPath string, startTime, endTime *time.Time) ([]Event, error) {
	if err := r.wait(ctx); err != nil {
		return nil, err
	}
	return r.inner.SearchEvents(ctx, calendarPath, startTime, endTime)
}

func (r *RateLimitedClient) CreateEvent(ctx context.Context, calendarPath string, event *Event) (string, error) {
	if err := r.wait(ctx); err != nil {
		return "", err
	}
	return r.inner.CreateEvent(ctx, calendarPath, event)
}

func (r *RateLimitedClient) UpdateEvent(ctx context.Context, eventPath string, update *EventUpdate) error {
	if err := r.wait(ctx); err != nil {
		return err
	}
	return r.inner.UpdateEvent(ctx, eventPath, update)
}

func (r *RateLimitedClient) DeleteEvent(ctx context.Context, eventPath string) error {
	if err := r.wait(ctx); err != nil {
		return err
	}
	return r.inner.DeleteEvent(ctx, eventPath)
}

func (r *RateLimitedClient) GetEventPath(calendarPath, eventID string) string {
	return r.inner.GetEventPath(calendarPath, eventID)
}
