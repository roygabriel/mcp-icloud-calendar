package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

// SearchEventsHandler creates a handler for searching calendar events
func SearchEventsHandler(client *caldav.Client, defaultCalendar string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Get calendar path
		calendarID, _ := args["calendarId"].(string)
		if calendarID == "" {
			calendarID = defaultCalendar
		}

		if calendarID == "" {
			return mcp.NewToolResultError("calendarId is required (no default calendar configured)"), nil
		}

		// Parse optional time filters
		var startTime, endTime *time.Time

		if startStr, ok := args["startTime"].(string); ok && startStr != "" {
			t, err := time.Parse(time.RFC3339, startStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid startTime format: %v (use ISO 8601 format like '2024-01-15T14:30:00Z')", err)), nil
			}
			startTime = &t
		}

		if endStr, ok := args["endTime"].(string); ok && endStr != "" {
			t, err := time.Parse(time.RFC3339, endStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid endTime format: %v (use ISO 8601 format like '2024-01-15T14:30:00Z')", err)), nil
			}
			endTime = &t
		}

		// Search events
		events, err := client.SearchEvents(ctx, calendarID, startTime, endTime)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search events: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"count":  len(events),
			"events": events,
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
