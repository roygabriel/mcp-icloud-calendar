package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

func newUpdateRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "update_event",
			Arguments: args,
		},
	}
}

func TestUpdateEventHandler_HappyPath(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId": "event-123",
		"title":   "Updated Title",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}

	if mock.LastUpdateEvent.Title == nil || *mock.LastUpdateEvent.Title != "Updated Title" {
		t.Errorf("title not set correctly")
	}
}

func TestUpdateEventHandler_ClearDescription(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId":     "event-123",
		"description": "",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}

	if mock.LastUpdateEvent.Description == nil {
		t.Fatal("description should be non-nil (set to empty = clear)")
	}
	if *mock.LastUpdateEvent.Description != "" {
		t.Errorf("description = %q, want empty string (clear)", *mock.LastUpdateEvent.Description)
	}
}

func TestUpdateEventHandler_MissingEventID(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing eventId")
	}
}

func TestUpdateEventHandler_InvalidTimeFormat(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId":   "event-123",
		"startTime": "not-a-time",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid time format")
	}
}

func TestUpdateEventHandler_CalDAVError(t *testing.T) {
	mock := &caldav.MockClient{
		UpdateEventErr: fmt.Errorf("not found"),
	}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId": "event-123",
		"title":   "New Title",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestUpdateEventHandler_InvalidEventID(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId": "../../../etc/passwd",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for path traversal in eventId")
	}
}
