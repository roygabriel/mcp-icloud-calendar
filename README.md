# iCloud Calendar MCP Server

A [Model Context Protocol](https://modelcontextprotocol.io) server that gives AI assistants full access to Apple iCloud Calendar through CalDAV. List calendars, search events, create, update, and delete events -- all from Claude or any MCP-compatible client.

Built with Go and the [mcp-go SDK](https://mcp-go.dev). Ships as a single static binary for Linux, macOS, and Windows.

## Features

- **Calendar Operations** - List, search, create, update, and delete iCloud calendar events
- **Recurring Events** - Expand RRULE recurring events into individual occurrences within a date range
- **Attendee Management** - Add attendees with roles (CHAIR, REQ-PARTICIPANT, OPT-PARTICIPANT) and statuses
- **Multi-Account Support** - Manage multiple iCloud accounts from a single server instance
- **Circuit Breaker** - Automatic fail-fast after repeated upstream failures, with self-healing recovery
- **Concurrency Control** - Bounded concurrent tool execution to prevent resource exhaustion
- **Secret Redaction** - Passwords are automatically scrubbed from all log output
- **Hardened Transport** - Granular HTTP timeouts (dial, TLS, response header) and response body size limits (10 MB)
- **Rate Limiting** - Per-account token bucket rate limiting to avoid iCloud throttling
- **Retry with Backoff** - Automatic exponential backoff on transient failures for idempotent operations
- **Structured Logging** - JSON-formatted logs with UUID correlation IDs, tool names, and durations via `slog`
- **Audit Logging** - Destructive operations logged with full arguments for traceability (no PII)
- **Health & Metrics** - Kubernetes-style `/healthz`, `/readyz` endpoints and Prometheus `/metrics`
- **Input Validation** - Path traversal, injection, type, and range checks on all parameters
- **MCP Annotations** - Tool annotations (read-only, destructive, idempotent) for client-side safety
- **Graceful Shutdown** - Signal handling on SIGTERM/SIGINT with readiness management
- **mTLS Support** - Optional client certificate authentication for enterprise deployments
- **Secure Credentials** - `file://` prefix for Docker/Kubernetes secret injection
- **CI Pipeline** - Tests, linting, coverage enforcement, and vulnerability scanning

## Prerequisites

- **Go 1.21+** -- [install](https://go.dev/doc/install) (only needed when building from source)
- **iCloud account** with two-factor authentication enabled
- **App-specific password** -- required for CalDAV access

### Generating an App-Specific Password

1. Go to [appleid.apple.com](https://appleid.apple.com) and sign in
2. Navigate to **Sign-In and Security** > **App-Specific Passwords**
3. Click **Generate an app-specific password**
4. Enter a label (e.g. "MCP Calendar Server") and click **Create**
5. Copy the generated password (`xxxx-xxxx-xxxx-xxxx`) and store it securely

Notes:
- Your Apple ID must have two-factor authentication enabled
- You can create up to 25 active app-specific passwords
- Changing your main Apple ID password revokes all app-specific passwords
- Never use your main iCloud password for CalDAV access

## Installation

### From Source

```bash
git clone https://github.com/rgabriel/mcp-icloud-calendar.git
cd mcp-icloud-calendar
make build
```

### Using `go install`

```bash
go install github.com/rgabriel/mcp-icloud-calendar@latest
```

### Docker

```bash
docker build -t mcp-icloud-calendar .

docker run \
  -e ICLOUD_EMAIL="you@icloud.com" \
  -e ICLOUD_PASSWORD="xxxx-xxxx-xxxx-xxxx" \
  mcp-icloud-calendar
```

The Docker image uses a multi-stage build with a [distroless](https://github.com/GoogleContainerTools/distroless) base image and runs as a non-root user.

### Prebuilt Binaries

Download the binary for your platform from the [Releases](https://github.com/rgabriel/mcp-icloud-calendar/releases) page.

| Platform | Architecture | Binary |
|----------|-------------|--------|
| Linux | x86_64 | `mcp-icloud-calendar-linux-amd64` |
| Linux | ARM64 | `mcp-icloud-calendar-linux-arm64` |
| macOS | Intel | `mcp-icloud-calendar-macos-amd64` |
| macOS | Apple Silicon | `mcp-icloud-calendar-macos-arm64` |
| Windows | x86_64 | `mcp-icloud-calendar-windows-amd64.exe` |

SHA256 checksums are provided alongside each binary.

## Configuration

### Single Account

The server requires two environment variables at minimum:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ICLOUD_EMAIL` | Yes | | Your iCloud email address (Apple ID) |
| `ICLOUD_PASSWORD` | Yes | | App-specific password from appleid.apple.com |
| `ICLOUD_CALENDAR_ID` | No | | Default calendar path (e.g., `/1234567/calendars/home/`) |
| `LOG_LEVEL` | No | `INFO` | Logging verbosity: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `TOOL_TIMEOUT` | No | `25s` | Timeout per tool call (Go duration, e.g., `30s`, `1m`) |
| `MAX_RETRIES` | No | `3` | Retry attempts for transient CalDAV failures |
| `RETRY_BASE_DELAY` | No | `1s` | Base delay for exponential backoff |
| `RATE_LIMIT_RPS` | No | `10` | CalDAV requests per second per account |
| `RATE_LIMIT_BURST` | No | `20` | Burst allowance for rate limiter |
| `MAX_CONNS_PER_HOST` | No | `10` | Max HTTP connections to iCloud per account |
| `CB_THRESHOLD` | No | `5` | Consecutive failures before circuit breaker opens (1-100) |
| `CB_RESET_TIMEOUT` | No | `30s` | Time before circuit breaker probes again (1s-5m) |
| `MAX_CONCURRENT` | No | `10` | Maximum concurrent tool calls (1-1000) |
| `HEALTH_PORT` | No | | Port for health/metrics HTTP server (e.g., `8080`) |
| `TLS_CERT_FILE` | No | | Client TLS certificate for mTLS |
| `TLS_KEY_FILE` | No | | Client TLS key for mTLS |
| `TLS_CA_FILE` | No | | Custom CA certificate |

You can set these as environment variables or place them in a `.env` file:

```bash
cp .env.example .env
# Edit .env with your credentials
```

Credentials support `file://` prefixes for Docker/Kubernetes secrets (e.g., `ICLOUD_PASSWORD=file:///run/secrets/password`).

### Multi-Account

To manage multiple iCloud accounts, set the `ACCOUNTS_FILE` environment variable pointing to a JSON file:

```json
{
  "accounts": [
    {
      "name": "personal",
      "email": "personal@icloud.com",
      "password": "xxxx-xxxx-xxxx-xxxx",
      "calendarId": "/1234567/calendars/home/"
    },
    {
      "name": "work",
      "email": "work@icloud.com",
      "password": "yyyy-yyyy-yyyy-yyyy"
    }
  ]
}
```

Each tool accepts an optional `account` parameter. Omit it to use the default account.

## Usage with Claude Desktop

Add the server to your Claude Desktop configuration file.

**macOS** -- `~/Library/Application Support/Claude/claude_desktop_config.json`

**Linux** -- `~/.config/claude/claude_desktop_config.json`

**Windows** -- `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "icloud-calendar": {
      "command": "/path/to/mcp-icloud-calendar",
      "env": {
        "ICLOUD_EMAIL": "you@icloud.com",
        "ICLOUD_PASSWORD": "xxxx-xxxx-xxxx-xxxx"
      }
    }
  }
}
```

Restart Claude Desktop after saving.

## Usage with Claude Code CLI

Add the server using the `claude mcp add` command:

```bash
claude mcp add icloud-calendar -- /path/to/mcp-icloud-calendar
```

Set the credentials as environment variables before launching Claude Code, or export them in your shell profile:

```bash
export ICLOUD_EMAIL="you@icloud.com"
export ICLOUD_PASSWORD="xxxx-xxxx-xxxx-xxxx"
```

Alternatively, add with explicit environment variables using a wrapper script:

```bash
claude mcp add icloud-calendar -- env ICLOUD_EMAIL=you@icloud.com ICLOUD_PASSWORD=xxxx-xxxx-xxxx-xxxx /path/to/mcp-icloud-calendar
```

To verify the server is registered:

```bash
claude mcp list
```

## Available Tools

The server exposes 5 MCP tools. Each tool includes schema constraints and annotations indicating whether it is read-only, destructive, or idempotent.

### list_calendars

List all available iCloud calendars. Returns each calendar's path, display name, description, and color. Call this first to discover valid `calendarId` values.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `account` | string | | Account name for multi-account setups |

**Example Request:**
```json
{}
```

**Example Response:**
```json
{
  "count": 2,
  "calendars": [
    {
      "path": "/1234567/calendars/home/",
      "name": "Home",
      "description": "Personal calendar",
      "color": ""
    },
    {
      "path": "/1234567/calendars/work/",
      "name": "Work",
      "description": "",
      "color": ""
    }
  ]
}
```

### search_events

Search for calendar events within a date range. Returns paginated results with event details including recurrence info and attendees.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `account` | string | | Account name for multi-account setups |
| `calendarId` | string | *(server default)* | Calendar path from `list_calendars` |
| `startTime` | string | | Start of date range (RFC 3339, e.g., `2025-03-01T00:00:00Z`) |
| `endTime` | string | | End of date range (RFC 3339) |
| `limit` | number | `50` | Max events to return (1-500) |
| `offset` | number | `0` | Events to skip for pagination |
| `expandRecurrence` | boolean | `false` | Expand recurring events into individual occurrences (requires both `startTime` and `endTime`) |

**Example Request:**
```json
{
  "calendarId": "/1234567/calendars/home/",
  "startTime": "2025-03-01T00:00:00Z",
  "endTime": "2025-03-31T23:59:59Z",
  "limit": 10
}
```

**Example Response:**
```json
{
  "count": 2,
  "total": 2,
  "offset": 0,
  "limit": 10,
  "events": [
    {
      "id": "abc123@mcp-icloud-calendar",
      "path": "/1234567/calendars/home/abc123.ics",
      "title": "Team Standup",
      "description": "Daily sync",
      "location": "Zoom",
      "startTime": "2025-03-15T09:00:00Z",
      "endTime": "2025-03-15T09:30:00Z",
      "recurrence": "FREQ=DAILY;COUNT=30",
      "timezone": "America/New_York",
      "attendees": [
        {"email": "alice@example.com", "name": "Alice", "role": "REQ-PARTICIPANT", "status": "ACCEPTED"}
      ]
    }
  ]
}
```

### create_event

Create a new calendar event. Returns the created event's unique ID.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `account` | string | | Account name for multi-account setups |
| `title` | string | *(required)* | Event title or summary |
| `startTime` | string | *(required)* | Start time (RFC 3339) |
| `endTime` | string | *(required)* | End time (RFC 3339) |
| `description` | string | | Event description or notes |
| `location` | string | | Event location |
| `calendarId` | string | *(server default)* | Calendar path to create the event in |
| `attendees` | string | | JSON array of attendee objects (see below) |

**Example Request:**
```json
{
  "title": "Project Review",
  "startTime": "2025-03-20T14:00:00Z",
  "endTime": "2025-03-20T15:00:00Z",
  "description": "Q1 project review meeting",
  "location": "Conference Room B",
  "calendarId": "/1234567/calendars/work/",
  "attendees": "[{\"email\":\"alice@example.com\",\"name\":\"Alice\",\"role\":\"REQ-PARTICIPANT\"}]"
}
```

**Example Response:**
```json
{
  "success": true,
  "eventId": "d4e5f6a7-b8c9-0123-4567-890abcdef012@mcp-icloud-calendar",
  "message": "Event 'Project Review' created successfully"
}
```

**Attendee format:**

```json
[
  {"email": "alice@example.com", "name": "Alice", "role": "REQ-PARTICIPANT"},
  {"email": "bob@example.com", "name": "Bob", "role": "OPT-PARTICIPANT", "status": "TENTATIVE"}
]
```

Supported roles: `CHAIR`, `REQ-PARTICIPANT`, `OPT-PARTICIPANT`. Supported statuses: `NEEDS-ACTION`, `ACCEPTED`, `DECLINED`, `TENTATIVE`.

### update_event

Update specific fields of an existing event. Only include the fields you want to change -- omitted fields remain unchanged.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `account` | string | | Account name for multi-account setups |
| `eventId` | string | *(required)* | Event ID (UID) from `search_events` |
| `calendarId` | string | *(server default)* | Calendar path containing the event |
| `title` | string | | Updated title |
| `description` | string | | Updated description |
| `location` | string | | Updated location |
| `startTime` | string | | Updated start time (RFC 3339) |
| `endTime` | string | | Updated end time (RFC 3339) |

**Example Request:**
```json
{
  "eventId": "abc123@mcp-icloud-calendar",
  "calendarId": "/1234567/calendars/work/",
  "title": "Updated Project Review",
  "location": "Conference Room A"
}
```

**Example Response:**
```json
{
  "success": true,
  "message": "Event updated successfully"
}
```

### delete_event

Permanently delete a calendar event. This action cannot be undone.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `account` | string | | Account name for multi-account setups |
| `eventId` | string | *(required)* | Event ID (UID) from `search_events` |
| `calendarId` | string | *(required)* | Calendar path containing the event |

**Example Request:**
```json
{
  "eventId": "abc123@mcp-icloud-calendar",
  "calendarId": "/1234567/calendars/work/"
}
```

**Example Response:**
```json
{
  "success": true,
  "message": "Event deleted successfully"
}
```

## Development

### Building

```bash
make build          # Build binary with version embedding
make test           # Run tests with race detector and coverage
make cover          # Generate HTML coverage report
make check-cover    # Verify 80% coverage threshold
make lint           # Run golangci-lint
make vuln           # Run govulncheck
make all            # Run lint, test, and vuln checks
make clean          # Remove build artifacts
make docker         # Build Docker image
make run            # Build and run
```

### Running Locally

```bash
export ICLOUD_EMAIL="you@icloud.com"
export ICLOUD_PASSWORD="xxxx-xxxx-xxxx-xxxx"
make run
```

### Testing

The project uses table-driven tests with mock implementations of the `CalendarService` interface -- no live CalDAV connection required. Tests cover all tool handlers, CalDAV client logic, input validation, retry/rate-limiting wrappers, circuit breaker state transitions, concurrency middleware, timeout middleware, middleware chaining, recurrence expansion, attendee parsing, secret redaction, and error paths.

```bash
make test
```

### Testing with MCP Inspector

Use the [MCP Inspector](https://github.com/modelcontextprotocol/inspector) to interactively test the server:

```bash
npx @modelcontextprotocol/inspector mcp-icloud-calendar
```

### CI Pipeline

Every push to `main` or `dev` and every pull request runs:

- `go vet` and `go test -race` -- correctness and data race detection
- `golangci-lint` -- static analysis (errcheck, govet, staticcheck, gosec, gocritic, and more)
- `govulncheck` -- known vulnerability scanning
- Coverage enforcement -- minimum 80% threshold

Tagged releases (`v*.*.*`) trigger automated cross-platform builds with SHA256 checksums.

## Architecture

```
mcp-icloud-calendar/
  main.go                Server setup, middleware chain, audit hook, graceful shutdown
  config/
    config.go            Environment variable loading, validation, file:// credential support
    accounts.go          Multi-account JSON configuration
  caldav/
    interface.go         CalendarService interface
    client.go            CalDAV client (caldav.icloud.com, TLS/mTLS, response body limits)
    retry.go             Retry wrapper with exponential backoff
    ratelimit.go         Rate-limiting wrapper (token bucket)
    circuitbreaker.go    Three-state circuit breaker (Closed/Open/HalfOpen)
    circuitbreaker_client.go  CalendarService wrapper with breaker logic
    recurrence.go        RRULE expansion for recurring events
    attendees.go         Attendee parsing and serialization
    validation.go        Input validation for CalDAV parameters
  tools/
    accounts.go          AccountClients multi-account resolver
    list_calendars.go    list_calendars handler
    search_events.go     search_events handler
    create_event.go      create_event handler
    update_event.go      update_event handler
    delete_event.go      delete_event handler
  health/server.go       Health check and readiness endpoints
  metrics/               Prometheus metrics and tool call middleware
  middleware/             Concurrency cap, timeout, request ID, chain composer
  logging/               Structured JSON logging with secret redaction
```

**Client chain:** Each account gets its own pipeline: `realClient → RateLimitedClient → CircuitBreakerClient → RetryClient`

**Middleware chain:** Each tool call passes through (outermost first):
1. **Concurrency cap** (bounded concurrent tool calls)
2. **Timeout** (configurable deadline per tool call)
3. **Request ID** (UUID correlation for log tracing)
4. **Metrics** (Prometheus duration and outcome tracking)

**Audit logging:** Mutating operations (`create_event`, `update_event`, `delete_event`) are logged via a post-call hook with tool name, account, calendar ID, event ID, full arguments, and status.

### Dependencies

| Package | Purpose |
|---------|---------|
| [mcp-go](https://github.com/mark3labs/mcp-go) | MCP SDK -- tool registration, stdio transport |
| [go-webdav](https://github.com/emersion/go-webdav) | CalDAV protocol client |
| [go-ical](https://github.com/emersion/go-ical) | iCalendar (RFC 5545) parsing |
| [rrule-go](https://github.com/teambition/rrule-go) | Recurrence rule expansion |
| [godotenv](https://github.com/joho/godotenv) | `.env` file loading |
| [uuid](https://github.com/google/uuid) | Event UID and request ID generation |
| [prometheus/client_golang](https://github.com/prometheus/client_golang) | Prometheus metrics |
| [x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) | Token bucket rate limiter |

## Security

- **App-specific passwords only** -- never accepts or stores your main iCloud password
- **Secret redaction** -- passwords are replaced with `[REDACTED]` in all log output (messages, attributes, nested groups)
- **TLS everywhere** -- all CalDAV communication uses HTTPS with TLS verification
- **mTLS support** -- optional client certificate authentication for enterprise environments
- **Hardened HTTP transport** -- dial timeout (5s), TLS handshake timeout (5s), response header timeout (5s), response body cap (10 MB)
- **Circuit breaker** -- prevents cascading failures when iCloud is unavailable
- **Concurrency cap** -- prevents resource exhaustion from unbounded parallel tool calls
- **Input validation** -- all tool parameters validated for type, range, path traversal, and injection
- **Credential file loading** -- `file://` prefix for secure secret injection (Docker, Kubernetes)
- **Distroless Docker image** -- minimal attack surface, runs as non-root
- **Audit trail** -- destructive operations logged with full arguments for compliance
- **No third-party data sharing** -- the server runs locally and communicates only with iCloud servers
- **Revocable access** -- app-specific passwords can be revoked at any time from appleid.apple.com

Never commit your `.env` file to version control. The `.gitignore` already excludes it.

## Troubleshooting

### Authentication Failed

- Verify you are using an app-specific password, not your main iCloud password
- Check that two-factor authentication is enabled on your Apple ID
- Regenerate a new app-specific password at appleid.apple.com
- Confirm your email address matches your Apple ID

### Calendar Not Found

- Run `list_calendars` to see the exact calendar paths your account has
- Calendar paths look like `/1234567/calendars/home/` -- always start with `/`
- Make sure you are using the `path` value from `list_calendars`, not the display name

### Invalid Date Format

- Use RFC 3339 / ISO 8601: `2025-01-15T14:30:00Z`
- Include timezone offset if not UTC: `2025-01-15T14:30:00-05:00`

### Timeouts or Slow Responses

- Check your internet connection
- Reduce the `limit` parameter for large result sets
- Use narrower date ranges with `startTime`/`endTime`
- Increase `TOOL_TIMEOUT` if your network is slow (default: 25s)
- The server enforces granular timeouts (5s dial, 5s TLS handshake, 5s response headers)
- Transient failures are retried automatically with exponential backoff (up to 3 attempts)

### Circuit Breaker Open

- If the circuit breaker opens after repeated failures, requests are rejected immediately with `circuit breaker is open`
- The breaker probes again after `CB_RESET_TIMEOUT` (default: 30s) -- no action needed
- Check iCloud service status if the breaker keeps re-opening
- Adjust `CB_THRESHOLD` (default: 5) to change sensitivity

### Concurrency Limit Reached

- If all concurrent slots are in use, new tool calls wait until a slot is available
- If the tool timeout expires while waiting, the call returns a deadline exceeded error
- Increase `MAX_CONCURRENT` (default: 10) if you need more parallel calls

### Recurring Event Not Expanding

- Set `expandRecurrence` to `true` in `search_events`
- Both `startTime` and `endTime` must be provided for recurrence expansion
- Expansion only works within the specified date range

### Event Not Found

- Verify the event ID matches a UID from `search_events`
- Ensure you are using the correct `calendarId`
- The event may have been deleted or moved since the ID was retrieved

## Contributing

Contributions are welcome. Please open an issue to discuss larger changes before submitting a pull request.

## License

MIT License -- see [LICENSE](LICENSE) for details.
