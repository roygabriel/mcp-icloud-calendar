# iCloud Calendar MCP Server

A Model Context Protocol (MCP) server that connects to Apple iCloud Calendar using the CalDAV protocol. This server enables AI assistants like Claude to interact with your iCloud calendars - list calendars, search events, create new events, update existing ones, and delete events.

Built with Go using the official [mcp-go SDK](https://mcp-go.dev) and works on all operating systems (Linux, Windows, macOS).

## Features

- **List Calendars** - Discover all available iCloud calendars with their IDs, names, and descriptions
- **Search Events** - Query calendar events with optional date range filters
- **Create Events** - Add new events with title, time, description, and location
- **Update Events** - Modify existing events by ID
- **Delete Events** - Remove events from calendars
- **Cross-Platform** - Works on Linux, Windows, and macOS
- **Secure** - Uses app-specific passwords (never your main iCloud password)

## Prerequisites

- **Go 1.21 or higher** - [Install Go](https://go.dev/doc/install)
- **iCloud Account** with two-factor authentication (2FA) enabled
- **App-Specific Password** - Required for CalDAV access (see setup below)

## App-Specific Password Setup

iCloud requires an app-specific password for third-party applications to access your calendar data. Follow these steps:

1. Go to [Apple ID Account Management](https://appleid.apple.com)
2. Sign in with your Apple ID
3. Navigate to **Sign-In and Security** section
4. Click on **App-Specific Passwords**
5. Click **Generate an app-specific password**
6. Enter a label like "MCP Calendar Server"
7. Click **Create**
8. Copy the generated password (format: `xxxx-xxxx-xxxx-xxxx`)
9. Save this password securely - you won't be able to see it again

**Important Notes:**
- Your Apple ID must have two-factor authentication enabled
- You can create up to 25 active app-specific passwords
- If you change your main Apple ID password, all app-specific passwords are revoked
- Never use your main iCloud password for CalDAV access

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/rgabriel/mcp-icloud-calendar.git
cd mcp-icloud-calendar

# Build the server
go build -o mcp-icloud-calendar

# Optional: Install to your PATH
go install
```

### Using go install

```bash
go install github.com/rgabriel/mcp-icloud-calendar@latest
```

## Configuration

Create a `.env` file in the same directory as the server executable (for local testing):

```bash
cp .env.example .env
```

Edit `.env` and add your credentials:

```bash
ICLOUD_EMAIL=your-email@icloud.com
ICLOUD_PASSWORD=xxxx-xxxx-xxxx-xxxx
ICLOUD_CALENDAR_ID=/12345678/calendars/home/
```

**Environment Variables:**

- `ICLOUD_EMAIL` (required) - Your iCloud email address (Apple ID)
- `ICLOUD_PASSWORD` (required) - App-specific password from appleid.apple.com
- `ICLOUD_CALENDAR_ID` (optional) - Default calendar ID/path to use

**Note:** You can discover your calendar IDs using the `list_calendars` tool after connecting.

## Usage with Claude Desktop

Add this server to your Claude Desktop configuration file:

### macOS

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "icloud-calendar": {
      "command": "/path/to/mcp-icloud-calendar",
      "env": {
        "ICLOUD_EMAIL": "your-email@icloud.com",
        "ICLOUD_PASSWORD": "xxxx-xxxx-xxxx-xxxx",
        "ICLOUD_CALENDAR_ID": "/12345678/calendars/home/"
      }
    }
  }
}
```

### Windows

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "icloud-calendar": {
      "command": "C:\\path\\to\\mcp-icloud-calendar.exe",
      "env": {
        "ICLOUD_EMAIL": "your-email@icloud.com",
        "ICLOUD_PASSWORD": "xxxx-xxxx-xxxx-xxxx",
        "ICLOUD_CALENDAR_ID": "/12345678/calendars/home/"
      }
    }
  }
}
```

### Linux

Edit `~/.config/claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "icloud-calendar": {
      "command": "/path/to/mcp-icloud-calendar",
      "env": {
        "ICLOUD_EMAIL": "your-email@icloud.com",
        "ICLOUD_PASSWORD": "xxxx-xxxx-xxxx-xxxx",
        "ICLOUD_CALENDAR_ID": "/12345678/calendars/home/"
      }
    }
  }
}
```

After adding the configuration, restart Claude Desktop.

## Available Tools

### 1. list_calendars

Lists all available calendars with their IDs, names, and descriptions.

**Parameters:** None

**Example Response:**
```json
{
  "count": 2,
  "calendars": [
    {
      "Path": "/12345678/calendars/home/",
      "Name": "Home",
      "Description": "Personal calendar"
    },
    {
      "Path": "/12345678/calendars/work/",
      "Name": "Work",
      "Description": "Work-related events"
    }
  ]
}
```

### 2. search_events

Search and list calendar events with optional date range filters.

**Parameters:**
- `calendarId` (optional) - Calendar ID/path to search in
- `startTime` (optional) - Start time filter in ISO 8601 format
- `endTime` (optional) - End time filter in ISO 8601 format

**Example:**
```json
{
  "calendarId": "/12345678/calendars/home/",
  "startTime": "2024-01-15T00:00:00Z",
  "endTime": "2024-01-31T23:59:59Z"
}
```

**Example Response:**
```json
{
  "count": 2,
  "events": [
    {
      "id": "unique-event-id-1",
      "path": "/12345678/calendars/home/unique-event-id-1.ics",
      "title": "Team Meeting",
      "description": "Weekly sync with the team",
      "location": "Conference Room A",
      "startTime": "2024-01-15T14:00:00Z",
      "endTime": "2024-01-15T15:00:00Z",
      "timezone": "America/New_York"
    }
  ]
}
```

### 3. create_event

Create a new calendar event.

**Parameters:**
- `title` (required) - Event title/summary
- `startTime` (required) - Event start time in ISO 8601 format
- `endTime` (required) - Event end time in ISO 8601 format
- `description` (optional) - Event description
- `location` (optional) - Event location
- `calendarId` (optional) - Calendar ID/path to create event in

**Example:**
```json
{
  "title": "Doctor Appointment",
  "startTime": "2024-02-10T10:00:00Z",
  "endTime": "2024-02-10T11:00:00Z",
  "description": "Annual checkup",
  "location": "Main Street Clinic",
  "calendarId": "/12345678/calendars/home/"
}
```

**Example Response:**
```json
{
  "success": true,
  "eventId": "1707559200000000000@mcp-icloud-calendar",
  "message": "Event 'Doctor Appointment' created successfully"
}
```

### 4. update_event

Update an existing calendar event.

**Parameters:**
- `eventId` (required) - Event ID (UID) to update
- `calendarId` (optional) - Calendar ID/path containing the event
- `title` (optional) - New event title
- `description` (optional) - New event description
- `location` (optional) - New event location
- `startTime` (optional) - New event start time in ISO 8601 format
- `endTime` (optional) - New event end time in ISO 8601 format

**Example:**
```json
{
  "eventId": "1707559200000000000@mcp-icloud-calendar",
  "calendarId": "/12345678/calendars/home/",
  "title": "Doctor Appointment - Rescheduled",
  "startTime": "2024-02-11T14:00:00Z",
  "endTime": "2024-02-11T15:00:00Z"
}
```

**Example Response:**
```json
{
  "success": true,
  "eventId": "1707559200000000000@mcp-icloud-calendar",
  "message": "Event updated successfully"
}
```

### 5. delete_event

Delete a calendar event.

**Parameters:**
- `eventId` (required) - Event ID (UID) to delete
- `calendarId` (required) - Calendar ID/path containing the event

**Example:**
```json
{
  "eventId": "1707559200000000000@mcp-icloud-calendar",
  "calendarId": "/12345678/calendars/home/"
}
```

**Example Response:**
```json
{
  "success": true,
  "eventId": "1707559200000000000@mcp-icloud-calendar",
  "message": "Event deleted successfully"
}
```

## Date/Time Format

All date/time parameters use **ISO 8601 format** (RFC3339 in Go):

- Format: `YYYY-MM-DDTHH:MM:SSZ`
- Examples:
  - `2024-01-15T14:30:00Z` (UTC)
  - `2024-01-15T14:30:00-05:00` (with timezone offset)
  - `2024-01-15T14:30:00+01:00` (with timezone offset)

The server handles timezone conversions automatically.

## Development

### Running Locally

```bash
# Set environment variables
export ICLOUD_EMAIL="your-email@icloud.com"
export ICLOUD_PASSWORD="xxxx-xxxx-xxxx-xxxx"
export ICLOUD_CALENDAR_ID="/12345678/calendars/home/"

# Run the server
go run main.go
```

### Building

```bash
# Build for your current platform
go build -o mcp-icloud-calendar

# Build for specific platforms
GOOS=linux GOARCH=amd64 go build -o mcp-icloud-calendar-linux
GOOS=darwin GOARCH=arm64 go build -o mcp-icloud-calendar-macos
GOOS=windows GOARCH=amd64 go build -o mcp-icloud-calendar-windows.exe
```

### Testing with MCP Inspector

Use the [MCP Inspector](https://github.com/modelcontextprotocol/inspector) to test the server:

```bash
npx @modelcontextprotocol/inspector mcp-icloud-calendar
```

## Troubleshooting

### Authentication Failed

**Problem:** "Failed to connect to iCloud CalDAV (check credentials)"

**Solutions:**
- Verify you're using an **app-specific password**, not your main iCloud password
- Check that your Apple ID has two-factor authentication enabled
- Regenerate a new app-specific password at appleid.apple.com
- Ensure your email address is correct (it should be your Apple ID)

### Calendar Not Found

**Problem:** "calendarId is required (no default calendar configured)"

**Solutions:**
- Run the `list_calendars` tool to see available calendar IDs
- Set the `ICLOUD_CALENDAR_ID` environment variable to your preferred calendar
- Always specify `calendarId` parameter in tool calls

### Invalid Date Format

**Problem:** "invalid startTime format"

**Solutions:**
- Use ISO 8601 format: `YYYY-MM-DDTHH:MM:SSZ`
- Example: `2024-01-15T14:30:00Z`
- Include timezone offset if not UTC: `2024-01-15T14:30:00-05:00`

### Network Timeouts

**Problem:** Connection timeouts or slow responses

**Solutions:**
- Check your internet connection
- iCloud CalDAV servers may be temporarily unavailable
- The server has a 30-second timeout - wait and retry
- Check if you can access icloud.com in your browser

### Event Not Found

**Problem:** "failed to get existing event" or "failed to delete event"

**Solutions:**
- Verify the event ID is correct
- Make sure you're using the correct calendar ID
- The event may have already been deleted
- Use `search_events` to find the correct event ID

## Architecture

The server consists of:

- **CalDAV Client** (`caldav/client.go`) - Handles iCloud CalDAV protocol communication
- **Configuration** (`config/config.go`) - Loads and validates environment variables
- **Tool Handlers** (`tools/*.go`) - Implements MCP tool logic for each operation
- **Main Server** (`main.go`) - MCP server initialization and tool registration

## Dependencies

- [mcp-go](https://github.com/mark3labs/mcp-go) v0.43.2 - Official MCP Go SDK
- [go-webdav](https://github.com/emersion/go-webdav) v0.7.0 - WebDAV/CalDAV client
- [godotenv](https://github.com/joho/godotenv) v1.5.1 - Environment variable loader

## Security Considerations

- Never commit your `.env` file to version control
- Use app-specific passwords, never your main iCloud password
- App-specific passwords can be revoked at any time from appleid.apple.com
- The server runs locally and doesn't send data to third parties
- All communication with iCloud uses HTTPS (TLS encryption)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues, questions, or feature requests, please open an issue on [GitHub](https://github.com/rgabriel/mcp-icloud-calendar/issues).

## Acknowledgments

- Built with the official [mcp-go SDK](https://mcp-go.dev)
- CalDAV implementation using [go-webdav](https://github.com/emersion/go-webdav)
- Follows the [Model Context Protocol](https://modelcontextprotocol.io) specification
