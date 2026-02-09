package caldav

import (
	"context"

	"github.com/emersion/go-ical"
	extcaldav "github.com/emersion/go-webdav/caldav"
)

// backend defines the interface for CalDAV protocol operations.
// This abstraction over the external go-webdav client enables unit testing.
type backend interface {
	FindCurrentUserPrincipal(ctx context.Context) (string, error)
	FindCalendarHomeSet(ctx context.Context, principal string) (string, error)
	FindCalendars(ctx context.Context, homeSet string) ([]extcaldav.Calendar, error)
	QueryCalendar(ctx context.Context, path string, query *extcaldav.CalendarQuery) ([]extcaldav.CalendarObject, error)
	PutCalendarObject(ctx context.Context, path string, cal *ical.Calendar) (*extcaldav.CalendarObject, error)
	GetCalendarObject(ctx context.Context, path string) (*extcaldav.CalendarObject, error)
	RemoveAll(ctx context.Context, path string) error
}

// Compile-time assertion that the real caldav.Client satisfies backend.
var _ backend = (*extcaldav.Client)(nil)
