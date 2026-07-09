package tools

import (
	"fmt"
	"strings"

	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

func eventPathFromReference(client caldav.CalendarService, calendarID, eventRef string) (string, error) {
	if strings.HasPrefix(eventRef, "/") {
		if err := caldav.ValidateCalendarPath(eventRef); err != nil {
			return "", err
		}

		calendarPrefix := strings.TrimSuffix(calendarID, "/") + "/"
		if !strings.HasPrefix(eventRef, calendarPrefix) {
			return "", fmt.Errorf("event path is not inside calendarId")
		}
		if !strings.HasSuffix(eventRef, ".ics") {
			return "", fmt.Errorf("event path must end with .ics")
		}

		return eventRef, nil
	}

	if err := caldav.ValidateEventID(eventRef); err != nil {
		return "", err
	}
	return client.GetEventPath(calendarID, eventRef), nil
}
