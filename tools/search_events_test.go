package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

func newSearchRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "search_events",
			Arguments: args,
		},
	}
}

func TestSearchEventsHandler_HappyPath(t *testing.T) {
	mock := &caldav.MockClient{
		Events: []caldav.Event{
			{ID: "e1", Title: "Meeting", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)},
		},
	}

	handler := SearchEventsHandler(testAccounts(mock, "/cal/default"))
	result, err := handler(context.Background(), newSearchRequest(map[string]interface{}{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["count"].(float64) != 1 {
		t.Errorf("count = %v, want 1", response["count"])
	}
	if response["total"].(float64) != 1 {
		t.Errorf("total = %v, want 1", response["total"])
	}
}

func TestSearchEventsHandler_MissingCalendar(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := SearchEventsHandler(testAccounts(mock, ""))
	result, err := handler(context.Background(), newSearchRequest(map[string]interface{}{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing calendar")
	}
}

func TestSearchEventsHandler_InvalidDates(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := SearchEventsHandler(testAccounts(mock, "/cal/default"))

	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{"invalid startTime", map[string]interface{}{"startTime": "not-a-date"}},
		{"invalid endTime", map[string]interface{}{"endTime": "not-a-date"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler(context.Background(), newSearchRequest(tt.args))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error for invalid date")
			}
		})
	}
}

func TestSearchEventsHandler_CalDAVError(t *testing.T) {
	mock := &caldav.MockClient{
		SearchEventsErr: fmt.Errorf("server unavailable"),
	}
	handler := SearchEventsHandler(testAccounts(mock, "/cal/default"))
	result, err := handler(context.Background(), newSearchRequest(map[string]interface{}{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestSearchEventsHandler_InvalidCalendarPath(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := SearchEventsHandler(testAccounts(mock, ""))
	result, err := handler(context.Background(), newSearchRequest(map[string]interface{}{
		"calendarId": "../etc/passwd",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for path traversal")
	}
}

func TestSearchEventsHandler_Pagination(t *testing.T) {
	events := make([]caldav.Event, 5)
	for i := range events {
		events[i] = caldav.Event{ID: fmt.Sprintf("e%d", i), Title: fmt.Sprintf("Event %d", i)}
	}
	mock := &caldav.MockClient{Events: events}
	handler := SearchEventsHandler(testAccounts(mock, "/cal/default"))

	t.Run("limit 2 offset 0", func(t *testing.T) {
		result, err := handler(context.Background(), newSearchRequest(map[string]interface{}{
			"limit":  float64(2),
			"offset": float64(0),
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if response["count"].(float64) != 2 {
			t.Errorf("count = %v, want 2", response["count"])
		}
		if response["total"].(float64) != 5 {
			t.Errorf("total = %v, want 5", response["total"])
		}
	})

	t.Run("limit 2 offset 3", func(t *testing.T) {
		result, err := handler(context.Background(), newSearchRequest(map[string]interface{}{
			"limit":  float64(2),
			"offset": float64(3),
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if response["count"].(float64) != 2 {
			t.Errorf("count = %v, want 2", response["count"])
		}
	})

	t.Run("offset beyond total", func(t *testing.T) {
		result, err := handler(context.Background(), newSearchRequest(map[string]interface{}{
			"limit":  float64(10),
			"offset": float64(100),
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if response["count"].(float64) != 0 {
			t.Errorf("count = %v, want 0", response["count"])
		}
	})
}

func TestSearchEventsHandler_MultiAccount(t *testing.T) {
	workMock := &caldav.MockClient{
		Events: []caldav.Event{
			{ID: "w1", Title: "Work Meeting"},
		},
	}
	personalMock := &caldav.MockClient{
		Events: []caldav.Event{
			{ID: "p1", Title: "Personal Event"},
			{ID: "p2", Title: "Personal Event 2"},
		},
	}
	ac := testMultiAccounts(
		map[string]caldav.CalendarService{"work": workMock, "personal": personalMock},
		map[string]string{"work": "/cal/work", "personal": "/cal/personal"},
	)

	handler := SearchEventsHandler(ac)

	// Search work account
	result, err := handler(context.Background(), newSearchRequest(map[string]interface{}{
		"account": "work",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["count"].(float64) != 1 {
		t.Errorf("work count = %v, want 1", response["count"])
	}

	// Search personal account
	result, err = handler(context.Background(), newSearchRequest(map[string]interface{}{
		"account": "personal",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["count"].(float64) != 2 {
		t.Errorf("personal count = %v, want 2", response["count"])
	}

	// Unknown account
	result, err = handler(context.Background(), newSearchRequest(map[string]interface{}{
		"account": "unknown",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for unknown account")
	}
}
