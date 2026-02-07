package caldav

import (
	"fmt"
	"strings"
)

// ValidateCalendarPath checks that a calendar path doesn't contain path traversal or injection characters.
func ValidateCalendarPath(path string) error {
	if path == "" {
		return fmt.Errorf("calendar path cannot be empty")
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("calendar path contains path traversal")
	}
	if strings.ContainsAny(path, "\x00\n\r") {
		return fmt.Errorf("calendar path contains invalid characters")
	}
	return nil
}

// ValidateEventID checks that an event ID doesn't contain path traversal or injection characters.
func ValidateEventID(id string) error {
	if id == "" {
		return fmt.Errorf("event ID cannot be empty")
	}
	if strings.Contains(id, "..") {
		return fmt.Errorf("event ID contains path traversal")
	}
	if strings.ContainsAny(id, "\x00\n\r/") {
		return fmt.Errorf("event ID contains invalid characters")
	}
	return nil
}
