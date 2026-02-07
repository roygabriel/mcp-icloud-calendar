package caldav

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRetryClient_SearchEvents_Retries(t *testing.T) {
	mock := &MockClient{
		SearchEventsErr: fmt.Errorf("connection reset"),
	}

	rc := NewRetryClient(mock, 2, 1*time.Millisecond)
	_, err := rc.SearchEvents(context.Background(), "/cal", nil, nil)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	// 1 initial + 2 retries = 3 calls
	if mock.SearchCallCount != 3 {
		t.Errorf("expected 3 calls, got %d", mock.SearchCallCount)
	}
}

func TestRetryClient_SearchEvents_SucceedsAfterRetry(t *testing.T) {
	callCount := 0
	mock := &MockClient{
		Events: []Event{{ID: "e1", Title: "Test"}},
	}

	// Wrap with a custom error-then-succeed pattern
	failOnce := &failOnceMock{inner: mock, failCount: 1}

	rc := NewRetryClient(failOnce, 2, 1*time.Millisecond)
	events, err := rc.SearchEvents(context.Background(), "/cal", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	_ = callCount
}

func TestRetryClient_CreateEvent_NoRetry(t *testing.T) {
	mock := &MockClient{
		CreateEventErr: fmt.Errorf("server error"),
	}

	rc := NewRetryClient(mock, 3, 1*time.Millisecond)
	_, err := rc.CreateEvent(context.Background(), "/cal", &Event{Title: "test"})
	if err == nil {
		t.Fatal("expected error from CreateEvent")
	}
	// CreateEvent should NOT retry
	if mock.CreateCallCount != 1 {
		t.Errorf("expected 1 call (no retry), got %d", mock.CreateCallCount)
	}
}

func TestRetryClient_DeleteEvent_Retries(t *testing.T) {
	mock := &MockClient{
		DeleteEventErr: fmt.Errorf("timeout"),
	}

	rc := NewRetryClient(mock, 2, 1*time.Millisecond)
	err := rc.DeleteEvent(context.Background(), "/cal/event.ics")
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if mock.DeleteCallCount != 3 {
		t.Errorf("expected 3 calls, got %d", mock.DeleteCallCount)
	}
}

func TestRetryClient_ContextCancellation(t *testing.T) {
	mock := &MockClient{
		SearchEventsErr: fmt.Errorf("keep failing"),
	}

	rc := NewRetryClient(mock, 10, 100*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := rc.SearchEvents(ctx, "/cal", nil, nil)
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

// failOnceMock wraps a CalendarService and fails the first N SearchEvents calls
type failOnceMock struct {
	inner     CalendarService
	failCount int
	calls     int
}

var _ CalendarService = (*failOnceMock)(nil)

func (f *failOnceMock) DiscoverCalendarHomeSet(ctx context.Context) (string, error) {
	return f.inner.DiscoverCalendarHomeSet(ctx)
}

func (f *failOnceMock) ListCalendars(ctx context.Context) ([]Calendar, error) {
	return f.inner.ListCalendars(ctx)
}

func (f *failOnceMock) SearchEvents(ctx context.Context, path string, start, end *time.Time) ([]Event, error) {
	f.calls++
	if f.calls <= f.failCount {
		return nil, fmt.Errorf("transient error (call %d)", f.calls)
	}
	return f.inner.SearchEvents(ctx, path, start, end)
}

func (f *failOnceMock) CreateEvent(ctx context.Context, path string, event *Event) (string, error) {
	return f.inner.CreateEvent(ctx, path, event)
}

func (f *failOnceMock) UpdateEvent(ctx context.Context, path string, update *EventUpdate) error {
	return f.inner.UpdateEvent(ctx, path, update)
}

func (f *failOnceMock) DeleteEvent(ctx context.Context, path string) error {
	return f.inner.DeleteEvent(ctx, path)
}

func (f *failOnceMock) GetEventPath(calendarPath, eventID string) string {
	return f.inner.GetEventPath(calendarPath, eventID)
}
