package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rgabriel/mcp-icloud-calendar/caldav"
	"github.com/rgabriel/mcp-icloud-calendar/config"
	"github.com/rgabriel/mcp-icloud-calendar/health"
	"github.com/rgabriel/mcp-icloud-calendar/logging"
	"github.com/rgabriel/mcp-icloud-calendar/metrics"
	mw "github.com/rgabriel/mcp-icloud-calendar/middleware"
	"github.com/rgabriel/mcp-icloud-calendar/tools"
)

var version = "dev"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Load accounts (single or multi-account mode)
	accounts, err := config.LoadAccounts(cfg)
	if err != nil {
		log.Fatalf("failed to load accounts: %v", err)
	}

	// Collect secrets for redaction and initialize logging
	secrets := make([]string, 0, len(accounts))
	for _, acct := range accounts {
		secrets = append(secrets, acct.Password)
	}
	logging.SetupWithRedaction(cfg.LogLevel, secrets)

	// Create shared circuit breaker for CalDAV upstream
	breaker := caldav.NewCircuitBreaker(caldav.CircuitBreakerOptions{
		Threshold:    cfg.CBThreshold,
		ResetTimeout: cfg.CBResetTimeout,
	})

	// Create a CalendarService client per account, each with rate limiter + circuit breaker + retry
	clients := make(map[string]caldav.CalendarService, len(accounts))
	defaultCalendars := make(map[string]string, len(accounts))

	for name, acct := range accounts {
		caldavClient, err := caldav.NewClient(acct.Email, acct.Password, caldav.ClientOptions{
			MaxConnsPerHost: cfg.MaxConnsPerHost,
			TLSCertFile:     cfg.TLSCertFile,
			TLSKeyFile:      cfg.TLSKeyFile,
			TLSCAFile:       cfg.TLSCAFile,
		})
		if err != nil {
			slog.Error("failed to create CalDAV client", "account", name, "error", err)
			os.Exit(1)
		}

		// Validate connection
		ctx := context.Background()
		_, err = caldavClient.DiscoverCalendarHomeSet(ctx)
		if err != nil {
			slog.Error("failed to connect to iCloud CalDAV (check credentials)", "account", name, "error", err)
			os.Exit(1)
		}

		// Wrap: real -> rateLimited -> circuitBreaker -> retry
		clients[name] = caldav.NewRetryClient(
			caldav.NewCircuitBreakerClient(
				caldav.NewRateLimitedClient(caldavClient, cfg.RateLimitRPS, cfg.RateLimitBurst),
				breaker,
			),
			cfg.MaxRetries, cfg.RetryBaseDelay,
		)
		defaultCalendars[name] = acct.CalendarID

		slog.Info("account initialized", "account", name, "email", acct.Email)
	}

	accountClients := tools.NewAccountClients(clients, defaultCalendars)

	// Audit logging hook for mutating operations
	destructiveTools := map[string]bool{
		"create_event": true,
		"update_event": true,
		"delete_event": true,
	}
	auditHook := &server.Hooks{}
	auditHook.AddAfterCallTool(func(_ context.Context, _ any, req *mcp.CallToolRequest, result *mcp.CallToolResult) {
		toolName := req.Params.Name
		if !destructiveTools[toolName] {
			return
		}
		args := req.GetArguments()
		status := "success"
		if result != nil && result.IsError {
			status = "error"
		}
		slog.Info("audit",
			"audit", true,
			"tool", toolName,
			"account", args["account"],
			"calendarId", args["calendarId"],
			"eventId", args["eventId"],
			"args", args,
			"status", status,
		)
	})

	// Create MCP server with composed middleware chain
	s := server.NewMCPServer(
		"iCloud Calendar Server",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithHooks(auditHook),
		server.WithToolHandlerMiddleware(mw.Chain(
			mw.ConcurrencyMiddleware(cfg.MaxConcurrent),
			mw.TimeoutMiddleware(cfg.ToolTimeout),
			mw.RequestIDMiddleware(),
			metrics.ToolCallMiddleware(),
		)),
	)

	// Register search_events tool
	searchEventsTool := mcp.NewTool("search_events",
		mcp.WithDescription("Search for calendar events within a date range. Returns paginated results with event id, title, description, location, startTime, endTime, recurrence, timezone, and attendees. Use list_calendars first to discover valid calendarId values."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("account",
			mcp.Description("Account name for multi-account setups. Omit to use the default account."),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar path from list_calendars (e.g., '/1234567/calendars/ABCDEF-1234-5678/'). Uses the server's default calendar if omitted."),
		),
		mcp.WithString("startTime",
			mcp.Description("Start of date range filter in RFC 3339 format (e.g., '2025-03-01T00:00:00Z'). Events starting at or after this time are included."),
		),
		mcp.WithString("endTime",
			mcp.Description("End of date range filter in RFC 3339 format (e.g., '2025-03-31T23:59:59Z'). Events starting before this time are included."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of events to return per page."),
			mcp.DefaultNumber(50),
			mcp.Min(1),
			mcp.Max(500),
		),
		mcp.WithNumber("offset",
			mcp.Description("Number of events to skip for pagination. Use with limit to page through results."),
			mcp.DefaultNumber(0),
			mcp.Min(0),
		),
		mcp.WithBoolean("expandRecurrence",
			mcp.Description("When true, recurring events are expanded into individual occurrences within the startTime/endTime range. Requires both startTime and endTime to be set."),
			mcp.DefaultBool(false),
		),
	)
	s.AddTool(searchEventsTool, tools.SearchEventsHandler(accountClients))

	// Register create_event tool
	createEventTool := mcp.NewTool("create_event",
		mcp.WithDescription("Create a new calendar event on iCloud. Returns the created event's unique ID on success. Use list_calendars first to discover valid calendarId values."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithString("account",
			mcp.Description("Account name for multi-account setups. Omit to use the default account."),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Event title or summary displayed on the calendar."),
			mcp.MinLength(1),
		),
		mcp.WithString("startTime",
			mcp.Required(),
			mcp.Description("Event start time in RFC 3339 format (e.g., '2025-03-15T14:30:00Z'). Must be before endTime."),
		),
		mcp.WithString("endTime",
			mcp.Required(),
			mcp.Description("Event end time in RFC 3339 format (e.g., '2025-03-15T16:30:00Z'). Must be after startTime."),
		),
		mcp.WithString("description",
			mcp.Description("Detailed event description or notes."),
		),
		mcp.WithString("location",
			mcp.Description("Event location (e.g., 'Conference Room B', '123 Main St, City')."),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar path from list_calendars to create the event in. Uses the server's default calendar if omitted."),
		),
		mcp.WithString("attendees",
			mcp.Description("JSON array of attendee objects. Each object requires 'email' and optionally 'name', 'role' (CHAIR, REQ-PARTICIPANT, OPT-PARTICIPANT), and 'status' (NEEDS-ACTION, ACCEPTED, DECLINED, TENTATIVE). Example: [{\"email\":\"alice@example.com\",\"name\":\"Alice\"}]"),
		),
	)
	s.AddTool(createEventTool, tools.CreateEventHandler(accountClients))

	// Register update_event tool
	updateEventTool := mcp.NewTool("update_event",
		mcp.WithDescription("Update specific fields of an existing calendar event. Only include the fields you want to change. Omitted fields remain unchanged. Set a field to an empty string to clear it. Use search_events first to find the event's id."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("account",
			mcp.Description("Account name for multi-account setups. Omit to use the default account."),
		),
		mcp.WithString("eventId",
			mcp.Required(),
			mcp.Description("Unique event ID (UID) from a previous search_events result."),
		),
		mcp.WithString("calendarId",
			mcp.Description("Calendar path containing the event. Uses the server's default calendar if omitted."),
		),
		mcp.WithString("title",
			mcp.Description("Updated event title. Omit to keep the current title. Set to empty string to clear."),
		),
		mcp.WithString("description",
			mcp.Description("Updated event description. Omit to keep the current description. Set to empty string to clear."),
		),
		mcp.WithString("location",
			mcp.Description("Updated event location. Omit to keep the current location. Set to empty string to clear."),
		),
		mcp.WithString("startTime",
			mcp.Description("Updated start time in RFC 3339 format (e.g., '2025-03-15T14:30:00Z'). Omit to keep the current start time."),
		),
		mcp.WithString("endTime",
			mcp.Description("Updated end time in RFC 3339 format (e.g., '2025-03-15T16:30:00Z'). Omit to keep the current end time. Must be after startTime if both are provided."),
		),
	)
	s.AddTool(updateEventTool, tools.UpdateEventHandler(accountClients))

	// Register delete_event tool
	deleteEventTool := mcp.NewTool("delete_event",
		mcp.WithDescription("Permanently delete a calendar event. This action cannot be undone. Use search_events first to find the event's id and calendarId."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("account",
			mcp.Description("Account name for multi-account setups. Omit to use the default account."),
		),
		mcp.WithString("eventId",
			mcp.Required(),
			mcp.Description("Unique event ID (UID) from a previous search_events result."),
		),
		mcp.WithString("calendarId",
			mcp.Required(),
			mcp.Description("Calendar path containing the event. Required for delete operations to ensure the correct calendar is targeted."),
		),
	)
	s.AddTool(deleteEventTool, tools.DeleteEventHandler(accountClients))

	// Register list_calendars tool
	listCalendarsTool := mcp.NewTool("list_calendars",
		mcp.WithDescription("List all available iCloud calendars for the account. Returns each calendar's path (use as calendarId in other tools), display name, description, and color. Call this first to discover valid calendarId values before using search_events, create_event, update_event, or delete_event."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithString("account",
			mcp.Description("Account name for multi-account setups. Omit to use the default account."),
		),
	)
	s.AddTool(listCalendarsTool, tools.ListCalendarsHandler(accountClients))

	// Start health server if configured
	var healthServer *health.Server
	var httpServer *http.Server
	if cfg.HealthPort != "" {
		healthServer = health.NewServer()
		healthServer.Mux().Handle("/metrics", promhttp.Handler())
		httpServer = &http.Server{
			Addr:              ":" + cfg.HealthPort,
			Handler:           healthServer.Mux(),
			ReadHeaderTimeout: cfg.ToolTimeout,
		}
		go func() {
			slog.Info("health server starting", "port", cfg.HealthPort)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("health server error", "error", err)
			}
		}()
		healthServer.SetReady(true)
	}

	// Log startup
	slog.Info("server starting",
		"version", version,
		"accounts", len(accounts),
	)

	// Graceful shutdown with signal handling
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		slog.Info("received signal, shutting down", "signal", sig)
		if healthServer != nil {
			healthServer.SetReady(false)
		}
		if httpServer != nil {
			_ = httpServer.Close()
		}
		cancel()
	}()

	stdioServer := server.NewStdioServer(s)
	err = stdioServer.Listen(ctx, os.Stdin, os.Stdout)
	cancel()
	if err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("server shut down gracefully")
}
