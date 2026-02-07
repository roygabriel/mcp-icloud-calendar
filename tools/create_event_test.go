package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

func newCreateRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "create_event",
			Arguments: args,
		},
	}
}

func TestCreateEventHandler_HappyPath(t *testing.T) {
	mock := &caldav.MockClient{CreatedEventID: "new-uid-123"}
	handler := CreateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newCreateRequest(map[string]interface{}{
		"title":     "Team Meeting",
		"startTime": "2024-01-15T14:30:00Z",
		"endTime":   "2024-01-15T16:30:00Z",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}

	var response map[string]interface{}
	json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response)
	if response["eventId"] != "new-uid-123" {
		t.Errorf("eventId = %v, want new-uid-123", response["eventId"])
	}
}

func TestCreateEventHandler_MissingRequiredFields(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := CreateEventHandler(testAccounts(mock, "/cal/default"))

	tests := []struct {
		name string
		args map[string]interface{}
	}{
		{"missing title", map[string]interface{}{
			"startTime": "2024-01-15T14:30:00Z",
			"endTime":   "2024-01-15T16:30:00Z",
		}},
		{"missing startTime", map[string]interface{}{
			"title":   "Test",
			"endTime": "2024-01-15T16:30:00Z",
		}},
		{"missing endTime", map[string]interface{}{
			"title":     "Test",
			"startTime": "2024-01-15T14:30:00Z",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler(context.Background(), newCreateRequest(tt.args))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.IsError {
				t.Fatal("expected error for missing field")
			}
		})
	}
}

func TestCreateEventHandler_EndBeforeStart(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := CreateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newCreateRequest(map[string]interface{}{
		"title":     "Test",
		"startTime": "2024-01-15T16:30:00Z",
		"endTime":   "2024-01-15T14:30:00Z",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for end before start")
	}
}

func TestCreateEventHandler_CalDAVError(t *testing.T) {
	mock := &caldav.MockClient{
		CreateEventErr: fmt.Errorf("quota exceeded"),
	}
	handler := CreateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newCreateRequest(map[string]interface{}{
		"title":     "Test",
		"startTime": "2024-01-15T14:30:00Z",
		"endTime":   "2024-01-15T16:30:00Z",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestCreateEventHandler_MissingCalendar(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := CreateEventHandler(testAccounts(mock, ""))

	result, err := handler(context.Background(), newCreateRequest(map[string]interface{}{
		"title":     "Test",
		"startTime": "2024-01-15T14:30:00Z",
		"endTime":   "2024-01-15T16:30:00Z",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing calendar")
	}
}
