package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

func newDeleteRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "delete_event",
			Arguments: args,
		},
	}
}

func TestDeleteEventHandler_HappyPath(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := DeleteEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newDeleteRequest(map[string]interface{}{
		"eventId":    "event-123",
		"calendarId": "/cal/default",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}

	expectedPath := "/cal/default/event-123.ics"
	if mock.LastDeletePath != expectedPath {
		t.Errorf("delete path = %q, want %q", mock.LastDeletePath, expectedPath)
	}
}

func TestDeleteEventHandler_MissingFields(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := DeleteEventHandler(testAccounts(mock, ""))

	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{"missing eventId", map[string]interface{}{
			"calendarId": "/cal/default",
		}},
		{"missing calendarId", map[string]interface{}{
			"eventId": "event-123",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler(context.Background(), newDeleteRequest(tt.args))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error for missing field")
			}
		})
	}
}

func TestDeleteEventHandler_CalDAVError(t *testing.T) {
	mock := &caldav.MockClient{
		DeleteEventErr: fmt.Errorf("permission denied"),
	}
	handler := DeleteEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newDeleteRequest(map[string]interface{}{
		"eventId":    "event-123",
		"calendarId": "/cal/default",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestDeleteEventHandler_InvalidEventID(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := DeleteEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newDeleteRequest(map[string]interface{}{
		"eventId":    "../../../etc/passwd",
		"calendarId": "/cal/default",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for path traversal")
	}
}
