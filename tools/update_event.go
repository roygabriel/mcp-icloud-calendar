package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

// UpdateEventHandler creates a handler for updating calendar events
func UpdateEventHandler(client *caldav.Client, defaultCalendar string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Extract required parameters
		eventID, ok := args["eventId"].(string)
		if !ok || eventID == "" {
			return mcp.NewToolResultError("eventId is required"), nil
		}

		calendarID, _ := args["calendarId"].(string)
		if calendarID == "" {
			calendarID = defaultCalendar
		}

		if calendarID == "" {
			return mcp.NewToolResultError("calendarId is required (no default calendar configured)"), nil
		}

		// Build event path
		eventPath := client.GetEventPath(calendarID, eventID)

		// Create event update with optional fields
		event := &caldav.Event{}

		// Extract optional update fields
		if title, ok := args["title"].(string); ok && title != "" {
			event.Title = title
		}

		if description, ok := args["description"].(string); ok && description != "" {
			event.Description = description
		}

		if location, ok := args["location"].(string); ok && location != "" {
			event.Location = location
		}

		if startTimeStr, ok := args["startTime"].(string); ok && startTimeStr != "" {
			startTime, err := time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid startTime format: %v", err)), nil
			}
			event.StartTime = startTime
		}

		if endTimeStr, ok := args["endTime"].(string); ok && endTimeStr != "" {
			endTime, err := time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid endTime format: %v", err)), nil
			}
			event.EndTime = endTime
		}

		// Validate time order if both provided
		if !event.StartTime.IsZero() && !event.EndTime.IsZero() && event.EndTime.Before(event.StartTime) {
			return mcp.NewToolResultError("endTime must be after startTime"), nil
		}

		// Update event
		err := client.UpdateEvent(ctx, eventPath, event)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update event: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success": true,
			"eventId": eventID,
			"message": "Event updated successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
