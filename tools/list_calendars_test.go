package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

func TestListCalendarsHandler_HappyPath(t *testing.T) {
	mock := &caldav.MockClient{
		Calendars: []caldav.Calendar{
			{Path: "/cal/work", Name: "Work", Description: "Work calendar"},
			{Path: "/cal/personal", Name: "Personal", Description: ""},
		},
	}

	handler := ListCalendarsHandler(testAccounts(mock, ""))
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "list_calendars"},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	count := response["count"].(float64)
	if count != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

func TestListCalendarsHandler_CalDAVError(t *testing.T) {
	mock := &caldav.MockClient{
		ListCalendarsErr: fmt.Errorf("connection refused"),
	}

	handler := ListCalendarsHandler(testAccounts(mock, ""))
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "list_calendars"},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result")
	}
}

func TestListCalendarsHandler_MultiAccount(t *testing.T) {
	workMock := &caldav.MockClient{
		Calendars: []caldav.Calendar{
			{Path: "/cal/work", Name: "Work"},
		},
	}
	personalMock := &caldav.MockClient{
		Calendars: []caldav.Calendar{
			{Path: "/cal/p1", Name: "Personal 1"},
			{Path: "/cal/p2", Name: "Personal 2"},
		},
	}
	ac := testMultiAccounts(
		map[string]caldav.CalendarService{"work": workMock, "personal": personalMock},
		map[string]string{"work": "", "personal": ""},
	)
	handler := ListCalendarsHandler(ac)

	// List work calendars
	result, err := handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_calendars",
			Arguments: map[string]interface{}{"account": "work"},
		},
	})
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

	// List personal calendars
	result, err = handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "list_calendars",
			Arguments: map[string]interface{}{"account": "personal"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["count"].(float64) != 2 {
		t.Errorf("personal count = %v, want 2", response["count"])
	}
}
