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
func SearchEventsHandler(accounts *AccountClients) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		accountName, _ := args["account"].(string)
		client, defaultCalendar, err := accounts.Resolve(accountName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Get calendar path
		calendarID, _ := args["calendarId"].(string)
		if calendarID == "" {
			calendarID = defaultCalendar
		}

		if calendarID == "" {
			return mcp.NewToolResultError("calendarId is required (no default calendar configured)"), nil
		}

		if err := caldav.ValidateCalendarPath(calendarID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid calendarId: %v", err)), nil
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

		// Parse pagination params
		limit := 50
		if v, ok := args["limit"].(float64); ok && v > 0 {
			limit = int(v)
		}
		offset := 0
		if v, ok := args["offset"].(float64); ok && v >= 0 {
			offset = int(v)
		}

		expandRecurrence, _ := args["expandRecurrence"].(bool)

		// Search events
		events, err := client.SearchEvents(ctx, calendarID, startTime, endTime)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search events: %v", err)), nil
		}

		// Expand recurrences if requested
		if expandRecurrence && startTime != nil && endTime != nil {
			var expanded []caldav.Event
			for _, e := range events {
				if e.Recurrence != "" {
					occurrences, err := caldav.ExpandRecurrence(e, *startTime, *endTime)
					if err != nil {
						return mcp.NewToolResultError(fmt.Sprintf("failed to expand recurrence for event %s: %v", e.ID, err)), nil
					}
					expanded = append(expanded, occurrences...)
				} else {
					expanded = append(expanded, e)
				}
			}
			events = expanded
		}

		// Apply pagination
		total := len(events)
		if offset > total {
			offset = total
		}
		end := offset + limit
		if end > total {
			end = total
		}
		paginatedEvents := events[offset:end]

		// Format response
		response := map[string]interface{}{
			"count":  len(paginatedEvents),
			"total":  total,
			"offset": offset,
			"limit":  limit,
			"events": paginatedEvents,
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
