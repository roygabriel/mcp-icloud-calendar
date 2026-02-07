package caldav

import (
	"context"
	"time"
)

// MockClient implements CalendarService for testing.
type MockClient struct {
	Calendars      []Calendar
	Events         []Event
	CreatedEventID string
	Err            error
	// Per-method error overrides
	ListCalendarsErr error
	SearchEventsErr  error
	CreateEventErr   error
	UpdateEventErr   error
	DeleteEventErr   error
	DiscoverErr      error
	// Tracking
	LastUpdatePath  string
	LastUpdateEvent *EventUpdate
	LastDeletePath  string
	LastCreateEvent *Event
	CreateCallCount int
	DeleteCallCount int
	SearchCallCount int
}

var _ CalendarService = (*MockClient)(nil)

func (m *MockClient) DiscoverCalendarHomeSet(ctx context.Context) (string, error) {
	if m.DiscoverErr != nil {
		return "", m.DiscoverErr
	}
	if m.Err != nil {
		return "", m.Err
	}
	return "/calendars/user/", nil
}

func (m *MockClient) ListCalendars(ctx context.Context) ([]Calendar, error) {
	if m.ListCalendarsErr != nil {
		return nil, m.ListCalendarsErr
	}
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Calendars, nil
}

func (m *MockClient) SearchEvents(ctx context.Context, calendarPath string, startTime, endTime *time.Time) ([]Event, error) {
	m.SearchCallCount++
	if m.SearchEventsErr != nil {
		return nil, m.SearchEventsErr
	}
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Events, nil
}

func (m *MockClient) CreateEvent(ctx context.Context, calendarPath string, event *Event) (string, error) {
	m.CreateCallCount++
	m.LastCreateEvent = event
	if m.CreateEventErr != nil {
		return "", m.CreateEventErr
	}
	if m.Err != nil {
		return "", m.Err
	}
	id := m.CreatedEventID
	if id == "" {
		id = "mock-event-id"
	}
	return id, nil
}

func (m *MockClient) UpdateEvent(ctx context.Context, eventPath string, update *EventUpdate) error {
	m.LastUpdatePath = eventPath
	m.LastUpdateEvent = update
	if m.UpdateEventErr != nil {
		return m.UpdateEventErr
	}
	if m.Err != nil {
		return m.Err
	}
	return nil
}

func (m *MockClient) DeleteEvent(ctx context.Context, eventPath string) error {
	m.DeleteCallCount++
	m.LastDeletePath = eventPath
	if m.DeleteEventErr != nil {
		return m.DeleteEventErr
	}
	if m.Err != nil {
		return m.Err
	}
	return nil
}

func (m *MockClient) GetEventPath(calendarPath, eventID string) string {
	c := &Client{}
	return c.GetEventPath(calendarPath, eventID)
}
