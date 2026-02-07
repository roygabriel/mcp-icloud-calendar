package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

// CreateEventHandler creates a handler for creating calendar events
func CreateEventHandler(accounts *AccountClients) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		accountName, _ := args["account"].(string)
		client, defaultCalendar, err := accounts.Resolve(accountName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Extract required parameters
		title, ok := args["title"].(string)
		if !ok || title == "" {
			return mcp.NewToolResultError("title is required"), nil
		}

		startTimeStr, ok := args["startTime"].(string)
		if !ok || startTimeStr == "" {
			return mcp.NewToolResultError("startTime is required (ISO 8601 format like '2024-01-15T14:30:00Z')"), nil
		}

		endTimeStr, ok := args["endTime"].(string)
		if !ok || endTimeStr == "" {
			return mcp.NewToolResultError("endTime is required (ISO 8601 format like '2024-01-15T14:30:00Z')"), nil
		}

		// Parse times
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid startTime format: %v", err)), nil
		}

		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid endTime format: %v", err)), nil
		}

		// Validate time order
		if endTime.Before(startTime) {
			return mcp.NewToolResultError("endTime must be after startTime"), nil
		}

		// Extract optional parameters
		description, _ := args["description"].(string)
		location, _ := args["location"].(string)
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

		// Parse optional attendees
		var attendees []caldav.Attendee
		if attendeesStr, ok := args["attendees"].(string); ok && attendeesStr != "" {
			if err := json.Unmarshal([]byte(attendeesStr), &attendees); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid attendees JSON: %v", err)), nil
			}
		}

		// Create event
		event := &caldav.Event{
			Title:       title,
			Description: description,
			Location:    location,
			StartTime:   startTime,
			EndTime:     endTime,
			Attendees:   attendees,
		}

		eventID, err := client.CreateEvent(ctx, calendarID, event)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create event: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success": true,
			"eventId": eventID,
			"message": fmt.Sprintf("Event '%s' created successfully", title),
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
