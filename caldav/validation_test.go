package caldav

import (
	"testing"
)

func TestValidateCalendarPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid path", "/calendars/user/cal1", false},
		{"empty path", "", true},
		{"path traversal", "/calendars/../etc/passwd", true},
		{"null byte", "/calendars/cal\x00", true},
		{"newline", "/calendars/cal\n", true},
		{"carriage return", "/calendars/cal\r", true},
		{"valid with trailing slash", "/calendars/user/cal1/", false},
		{"just slash", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCalendarPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCalendarPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEventID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid with ics", "event123.ics", false},
		{"empty", "", true},
		{"path traversal", "../etc/passwd", true},
		{"null byte", "event\x00id", true},
		{"newline", "event\nid", true},
		{"slash", "path/event", true},
		{"valid alphanumeric", "abc123@mcp-icloud-calendar", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEventID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEventID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}
