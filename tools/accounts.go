// Package tools provides MCP tool handlers for iCloud Calendar operations
// including search, create, update, delete, and list calendars.
package tools

import (
	"fmt"
	"strings"

	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

// AccountClients maps account names to their CalendarService clients.
type AccountClients struct {
	clients          map[string]caldav.CalendarService
	defaultCalendars map[string]string // account name -> default calendar ID
}

// NewAccountClients creates an AccountClients from the given maps.
func NewAccountClients(clients map[string]caldav.CalendarService, defaultCalendars map[string]string) *AccountClients {
	return &AccountClients{
		clients:          clients,
		defaultCalendars: defaultCalendars,
	}
}

// Resolve returns the CalendarService and default calendar for the given account name.
// If accountName is empty, the "default" account is used.
func (a *AccountClients) Resolve(accountName string) (caldav.CalendarService, string, error) {
	if accountName == "" {
		accountName = "default"
	}

	client, ok := a.clients[accountName]
	if !ok {
		available := make([]string, 0, len(a.clients))
		for name := range a.clients {
			available = append(available, name)
		}
		return nil, "", fmt.Errorf("unknown account %q (available: %s)", accountName, strings.Join(available, ", "))
	}

	return client, a.defaultCalendars[accountName], nil
}

// AccountNames returns the list of available account names.
func (a *AccountClients) AccountNames() []string {
	names := make([]string, 0, len(a.clients))
	for name := range a.clients {
		names = append(names, name)
	}
	return names
}
