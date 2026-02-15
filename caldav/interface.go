// Package caldav provides a CalDAV client for iCloud Calendar with rate limiting,
// retry logic, and circuit breaker protection.
package caldav

import (
	"context"
	"time"
)

// CalendarService defines the interface for calendar operations.
type CalendarService interface {
	DiscoverCalendarHomeSet(ctx context.Context) (string, error)
	ListCalendars(ctx context.Context) ([]Calendar, error)
	SearchEvents(ctx context.Context, calendarPath string, startTime, endTime *time.Time) ([]Event, error)
	CreateEvent(ctx context.Context, calendarPath string, event *Event) (string, error)
	UpdateEvent(ctx context.Context, eventPath string, update *EventUpdate) error
	DeleteEvent(ctx context.Context, eventPath string) error
	GetEventPath(calendarPath, eventID string) string
}

// Compile-time assertion that Client implements CalendarService.
var _ CalendarService = (*Client)(nil)
