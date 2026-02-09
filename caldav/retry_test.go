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

func TestRetryClient_DiscoverCalendarHomeSet_Retries(t *testing.T) {
	mock := &MockClient{DiscoverErr: fmt.Errorf("discovery failed")}
	rc := NewRetryClient(mock, 1, 1*time.Millisecond)

	_, err := rc.DiscoverCalendarHomeSet(context.Background())
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
}

func TestRetryClient_DiscoverCalendarHomeSet_Success(t *testing.T) {
	mock := &MockClient{}
	rc := NewRetryClient(mock, 2, 1*time.Millisecond)

	homeSet, err := rc.DiscoverCalendarHomeSet(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if homeSet != "/calendars/user/" {
		t.Errorf("homeSet = %q, want /calendars/user/", homeSet)
	}
}

func TestRetryClient_ListCalendars_Retries(t *testing.T) {
	mock := &MockClient{ListCalendarsErr: fmt.Errorf("list failed")}
	rc := NewRetryClient(mock, 1, 1*time.Millisecond)

	_, err := rc.ListCalendars(context.Background())
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
}

func TestRetryClient_ListCalendars_Success(t *testing.T) {
	mock := &MockClient{
		Calendars: []Calendar{{Path: "/cal/1", Name: "Work"}},
	}
	rc := NewRetryClient(mock, 2, 1*time.Millisecond)

	cals, err := rc.ListCalendars(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cals) != 1 {
		t.Errorf("expected 1 calendar, got %d", len(cals))
	}
}

func TestRetryClient_UpdateEvent_NoRetry(t *testing.T) {
	mock := &MockClient{UpdateEventErr: fmt.Errorf("update failed")}
	rc := NewRetryClient(mock, 3, 1*time.Millisecond)

	title := "Updated"
	err := rc.UpdateEvent(context.Background(), "/cal/event.ics", &EventUpdate{Title: &title})
	if err == nil {
		t.Fatal("expected error from UpdateEvent")
	}
}

func TestRetryClient_UpdateEvent_Success(t *testing.T) {
	mock := &MockClient{}
	rc := NewRetryClient(mock, 2, 1*time.Millisecond)

	title := "Updated"
	err := rc.UpdateEvent(context.Background(), "/cal/event.ics", &EventUpdate{Title: &title})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetryClient_CreateEvent_Success(t *testing.T) {
	mock := &MockClient{CreatedEventID: "new-id"}
	rc := NewRetryClient(mock, 2, 1*time.Millisecond)

	id, err := rc.CreateEvent(context.Background(), "/cal", &Event{Title: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "new-id" {
		t.Errorf("id = %q, want new-id", id)
	}
}

func TestRetryClient_DeleteEvent_Success(t *testing.T) {
	mock := &MockClient{}
	rc := NewRetryClient(mock, 2, 1*time.Millisecond)

	err := rc.DeleteEvent(context.Background(), "/cal/event.ics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetryClient_GetEventPath(t *testing.T) {
	mock := &MockClient{}
	rc := NewRetryClient(mock, 2, 1*time.Millisecond)

	path := rc.GetEventPath("/cal/work", "event-123")
	expected := "/cal/work/event-123.ics"
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

func TestRetryClient_ZeroRetries(t *testing.T) {
	mock := &MockClient{SearchEventsErr: fmt.Errorf("fail")}
	rc := NewRetryClient(mock, 0, 1*time.Millisecond)

	_, err := rc.SearchEvents(context.Background(), "/cal", nil, nil)
	if err == nil {
		t.Fatal("expected error with zero retries")
	}
	if mock.SearchCallCount != 1 {
		t.Errorf("expected 1 call, got %d", mock.SearchCallCount)
	}
}

func TestRetryClient_SearchEvents_WithTimeParams(t *testing.T) {
	now := time.Now()
	end := now.Add(24 * time.Hour)
	mock := &MockClient{
		Events: []Event{{ID: "e1", Title: "Test"}},
	}
	rc := NewRetryClient(mock, 2, 1*time.Millisecond)

	events, err := rc.SearchEvents(context.Background(), "/cal", &now, &end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
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
