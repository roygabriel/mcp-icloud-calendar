package caldav

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRateLimitedClient_SearchEvents_Success(t *testing.T) {
	mock := &MockClient{
		Events: []Event{{ID: "e1", Title: "Test"}},
	}
	rl := NewRateLimitedClient(mock, 100, 10)

	events, err := rl.SearchEvents(context.Background(), "/cal", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestRateLimitedClient_SearchEvents_CancelledContext(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 0.001, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := rl.SearchEvents(ctx, "/cal", nil, nil)
	if err == nil {
		t.Fatal("expected error from rate limit with cancelled context")
	}
}

func TestRateLimitedClient_DiscoverCalendarHomeSet_Success(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 100, 10)

	homeSet, err := rl.DiscoverCalendarHomeSet(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if homeSet != "/calendars/user/" {
		t.Errorf("homeSet = %q, want /calendars/user/", homeSet)
	}
}

func TestRateLimitedClient_DiscoverCalendarHomeSet_CancelledContext(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 0.001, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := rl.DiscoverCalendarHomeSet(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRateLimitedClient_DiscoverCalendarHomeSet_InnerError(t *testing.T) {
	mock := &MockClient{DiscoverErr: fmt.Errorf("discovery failed")}
	rl := NewRateLimitedClient(mock, 100, 10)

	_, err := rl.DiscoverCalendarHomeSet(context.Background())
	if err == nil {
		t.Fatal("expected error from inner client")
	}
}

func TestRateLimitedClient_ListCalendars_Success(t *testing.T) {
	mock := &MockClient{
		Calendars: []Calendar{{Path: "/cal/1", Name: "Work"}},
	}
	rl := NewRateLimitedClient(mock, 100, 10)

	cals, err := rl.ListCalendars(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cals) != 1 {
		t.Errorf("expected 1 calendar, got %d", len(cals))
	}
}

func TestRateLimitedClient_ListCalendars_CancelledContext(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 0.001, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := rl.ListCalendars(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRateLimitedClient_ListCalendars_InnerError(t *testing.T) {
	mock := &MockClient{ListCalendarsErr: fmt.Errorf("list failed")}
	rl := NewRateLimitedClient(mock, 100, 10)

	_, err := rl.ListCalendars(context.Background())
	if err == nil {
		t.Fatal("expected error from inner client")
	}
}

func TestRateLimitedClient_CreateEvent_Success(t *testing.T) {
	mock := &MockClient{CreatedEventID: "new-id"}
	rl := NewRateLimitedClient(mock, 100, 10)

	id, err := rl.CreateEvent(context.Background(), "/cal", &Event{Title: "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "new-id" {
		t.Errorf("id = %q, want new-id", id)
	}
}

func TestRateLimitedClient_CreateEvent_CancelledContext(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 0.001, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := rl.CreateEvent(ctx, "/cal", &Event{Title: "Test"})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRateLimitedClient_CreateEvent_InnerError(t *testing.T) {
	mock := &MockClient{CreateEventErr: fmt.Errorf("create failed")}
	rl := NewRateLimitedClient(mock, 100, 10)

	_, err := rl.CreateEvent(context.Background(), "/cal", &Event{Title: "Test"})
	if err == nil {
		t.Fatal("expected error from inner client")
	}
}

func TestRateLimitedClient_UpdateEvent_Success(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 100, 10)

	title := "Updated"
	err := rl.UpdateEvent(context.Background(), "/cal/event.ics", &EventUpdate{Title: &title})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimitedClient_UpdateEvent_CancelledContext(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 0.001, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	title := "Updated"
	err := rl.UpdateEvent(ctx, "/cal/event.ics", &EventUpdate{Title: &title})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRateLimitedClient_UpdateEvent_InnerError(t *testing.T) {
	mock := &MockClient{UpdateEventErr: fmt.Errorf("update failed")}
	rl := NewRateLimitedClient(mock, 100, 10)

	title := "Updated"
	err := rl.UpdateEvent(context.Background(), "/cal/event.ics", &EventUpdate{Title: &title})
	if err == nil {
		t.Fatal("expected error from inner client")
	}
}

func TestRateLimitedClient_DeleteEvent_Success(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 100, 10)

	err := rl.DeleteEvent(context.Background(), "/cal/event.ics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimitedClient_DeleteEvent_CancelledContext(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 0.001, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := rl.DeleteEvent(ctx, "/cal/event.ics")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRateLimitedClient_DeleteEvent_InnerError(t *testing.T) {
	mock := &MockClient{DeleteEventErr: fmt.Errorf("delete failed")}
	rl := NewRateLimitedClient(mock, 100, 10)

	err := rl.DeleteEvent(context.Background(), "/cal/event.ics")
	if err == nil {
		t.Fatal("expected error from inner client")
	}
}

func TestRateLimitedClient_GetEventPath(t *testing.T) {
	mock := &MockClient{}
	rl := NewRateLimitedClient(mock, 100, 10)

	path := rl.GetEventPath("/cal/work", "event-123")
	expected := "/cal/work/event-123.ics"
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

func TestRateLimitedClient_SearchEvents_InnerError(t *testing.T) {
	mock := &MockClient{SearchEventsErr: fmt.Errorf("search failed")}
	rl := NewRateLimitedClient(mock, 100, 10)

	_, err := rl.SearchEvents(context.Background(), "/cal", nil, nil)
	if err == nil {
		t.Fatal("expected error from inner client")
	}
}
