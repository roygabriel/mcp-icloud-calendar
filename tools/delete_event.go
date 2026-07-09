package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

// DeleteEventHandler creates a handler for deleting calendar events
func DeleteEventHandler(accounts *AccountClients) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		// Delete event
		err = client.DeleteEvent(ctx, eventPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete event: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success": true,
			"eventId": eventID,
			"message": "Event deleted successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
