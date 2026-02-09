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

func TestUpdateEventHandler_InvalidEndTimeFormat(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId": "event-123",
		"endTime": "not-a-time",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid endTime format")
	}
}

func TestUpdateEventHandler_EndBeforeStart(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId":   "event-123",
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

func TestUpdateEventHandler_MissingCalendar(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, ""))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId": "event-123",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for missing calendar")
	}
}

func TestUpdateEventHandler_InvalidCalendarPath(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, ""))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId":    "event-123",
		"calendarId": "../etc/passwd",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for invalid calendarId")
	}
}

func TestUpdateEventHandler_SetLocation(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId":  "event-123",
		"location": "Room B",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if mock.LastUpdateEvent.Location == nil || *mock.LastUpdateEvent.Location != "Room B" {
		t.Error("location not set correctly")
	}
}

func TestUpdateEventHandler_UpdateTimes(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId":   "event-123",
		"startTime": "2024-02-01T10:00:00Z",
		"endTime":   "2024-02-01T11:00:00Z",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if mock.LastUpdateEvent.StartTime == nil {
		t.Fatal("startTime should be set")
	}
	if mock.LastUpdateEvent.EndTime == nil {
		t.Fatal("endTime should be set")
	}
}

func TestUpdateEventHandler_WithExplicitCalendarId(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, ""))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId":    "event-123",
		"calendarId": "/cal/explicit",
		"title":      "Updated",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success with explicit calendarId")
	}
}

func TestUpdateEventHandler_UnknownAccount(t *testing.T) {
	mock := &caldav.MockClient{}
	handler := UpdateEventHandler(testAccounts(mock, "/cal/default"))

	result, err := handler(context.Background(), newUpdateRequest(map[string]interface{}{
		"eventId": "event-123",
		"account": "nonexistent",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error for unknown account")
	}
}
