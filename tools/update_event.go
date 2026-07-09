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
func UpdateEventHandler(accounts *AccountClients) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		accountName, _ := args["account"].(string)
		client, defaultCalendar, err := accounts.Resolve(accountName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

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

		if err := caldav.ValidateCalendarPath(calendarID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid calendarId: %v", err)), nil
		}

		// Build event path. iCloud may store an event under a resource path that
		// differs from its UID, so callers may pass search_events.path as eventId.
		eventPath, err := eventPathFromReference(client, calendarID, eventID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid eventId: %v", err)), nil
		}

		// Build update with pointer fields
		update := &caldav.EventUpdate{}

		if title, exists := args["title"]; exists {
			if s, ok := title.(string); ok {
				update.Title = &s
			}
		}

		if description, exists := args["description"]; exists {
			if s, ok := description.(string); ok {
				update.Description = &s
			}
		}

		if location, exists := args["location"]; exists {
			if s, ok := location.(string); ok {
				update.Location = &s
			}
		}

		if startTimeStr, ok := args["startTime"].(string); ok && startTimeStr != "" {
			startTime, err := time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid startTime format: %v", err)), nil
			}
			update.StartTime = &startTime
		}

		if endTimeStr, ok := args["endTime"].(string); ok && endTimeStr != "" {
			endTime, err := time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid endTime format: %v", err)), nil
			}
			update.EndTime = &endTime
		}
		if travelTimeMinutes, exists, err := optionalNonNegativeInt(args, "travelTimeMinutes"); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid travelTimeMinutes: %v", err)), nil
		} else if exists {
			update.TravelTimeMinutes = &travelTimeMinutes
		}

		// Validate time order if both provided
		if update.StartTime != nil && update.EndTime != nil && update.EndTime.Before(*update.StartTime) {
			return mcp.NewToolResultError("endTime must be after startTime"), nil
		}

		// Update event
		err = client.UpdateEvent(ctx, eventPath, update)
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
