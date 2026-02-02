package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

// ListCalendarsHandler creates a handler for listing available calendars
func ListCalendarsHandler(client *caldav.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// List calendars
		calendars, err := client.ListCalendars(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list calendars: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"count":     len(calendars),
			"calendars": calendars,
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
