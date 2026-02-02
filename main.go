package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
	"github.com/rgabriel/mcp-icloud-calendar/config"
	"github.com/rgabriel/mcp-icloud-calendar/tools"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create CalDAV client
	caldavClient, err := caldav.NewClient(cfg.ICloudEmail, cfg.ICloudPassword)
	if err != nil {
		log.Fatalf("Failed to create CalDAV client: %v", err)
	}

	// Discover calendar home set to validate connection
	ctx := context.Background()
	_, err = caldavClient.DiscoverCalendarHomeSet(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to iCloud CalDAV (check credentials): %v", err)
	}

	// Create MCP server
	s := server.NewMCPServer(
		"iCloud Calendar Server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// Register search_events tool
	searchEventsTool := mcp.NewTool("search_events",
		mcp.WithDescription("Search and list calendar events with optional date range filters"),
		mcp.WithString("calendarId",
			mcp.Description("Calendar ID/path to search in (uses default if not specified)"),
		),
		mcp.WithString("startTime",
			mcp.Description("Optional start time filter in ISO 8601 format (e.g., '2024-01-15T14:30:00Z')"),
		),
		mcp.WithString("endTime",
			mcp.Description("Optional end time filter in ISO 8601 format (e.g., '2024-01-15T14:30:00Z')"),
		),
	)
	s.AddTool(searchEventsTool, tools.SearchEventsHandler(caldavClient, cfg.ICloudCalendarID))

	// Register create_event tool
	createEventTool := mcp.NewTool("create_event",
		mcp.WithDescription("Create a new calendar event"),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Event title/summary"),
		),
		mcp.WithString("startTime",
			mcp.Required(),
			mcp.Description("Event start time in ISO 8601 format (e.g., '2024-01-15T14:30:00Z')"),
		),
		mcp.WithString("endTime",
			mcp.Required(),
			mcp.Description("Event end time in ISO 8601 format (e.g., '2024-01-15T16:30:00Z')"),
		),
		mcp.WithString("description",
			mcp.Description("Optional event description"),
		),
		mcp.WithString("location",
			mcp.Description("Optional event location"),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar ID/path to create event in (uses default if not specified)"),
		),
	)
	s.AddTool(createEventTool, tools.CreateEventHandler(caldavClient, cfg.ICloudCalendarID))

	// Register update_event tool
	updateEventTool := mcp.NewTool("update_event",
		mcp.WithDescription("Update an existing calendar event"),
		mcp.WithString("eventId",
			mcp.Required(),
			mcp.Description("Event ID (UID) to update"),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar ID/path containing the event (uses default if not specified)"),
		),
		mcp.WithString("title",
			mcp.Description("New event title"),
		),
		mcp.WithString("description",
			mcp.Description("New event description"),
		),
		mcp.WithString("location",
			mcp.Description("New event location"),
		),
		mcp.WithString("startTime",
			mcp.Description("New event start time in ISO 8601 format"),
		),
		mcp.WithString("endTime",
			mcp.Description("New event end time in ISO 8601 format"),
		),
	)
	s.AddTool(updateEventTool, tools.UpdateEventHandler(caldavClient, cfg.ICloudCalendarID))

	// Register delete_event tool
	deleteEventTool := mcp.NewTool("delete_event",
		mcp.WithDescription("Delete a calendar event"),
		mcp.WithString("eventId",
			mcp.Required(),
			mcp.Description("Event ID (UID) to delete"),
		),
		mcp.WithString("calendarId",
			mcp.Required(),
			mcp.Description("Calendar ID/path containing the event"),
		),
	)
	s.AddTool(deleteEventTool, tools.DeleteEventHandler(caldavClient, cfg.ICloudCalendarID))

	// Register list_calendars tool
	listCalendarsTool := mcp.NewTool("list_calendars",
		mcp.WithDescription("List all available calendars with their IDs, names, and descriptions"),
	)
	s.AddTool(listCalendarsTool, tools.ListCalendarsHandler(caldavClient))

	// Log startup
	fmt.Fprintf(os.Stderr, "iCloud Calendar MCP Server v1.0.0 starting...\n")
	fmt.Fprintf(os.Stderr, "Connected to iCloud as: %s\n", cfg.ICloudEmail)
	if cfg.ICloudCalendarID != "" {
		fmt.Fprintf(os.Stderr, "Default calendar: %s\n", cfg.ICloudCalendarID)
	}

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
