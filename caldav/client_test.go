package caldav

import (
	"testing"
)

func TestGetEventPath(t *testing.T) {
	c := &Client{}

	tests := []struct {
		name         string
		calendarPath string
		eventID      string
		want         string
	}{
		{
			name:         "basic",
			calendarPath: "/calendars/user/cal1",
			eventID:      "event123",
			want:         "/calendars/user/cal1/event123.ics",
		},
		{
			name:         "trailing slash on calendar",
			calendarPath: "/calendars/user/cal1/",
			eventID:      "event123",
			want:         "/calendars/user/cal1/event123.ics",
		},
		{
			name:         "event already has .ics",
			calendarPath: "/calendars/user/cal1",
			eventID:      "event123.ics",
			want:         "/calendars/user/cal1/event123.ics",
		},
		{
			name:         "trailing slash and .ics",
			calendarPath: "/calendars/user/cal1/",
			eventID:      "event123.ics",
			want:         "/calendars/user/cal1/event123.ics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.GetEventPath(tt.calendarPath, tt.eventID)
			if got != tt.want {
				t.Errorf("GetEventPath(%q, %q) = %q, want %q", tt.calendarPath, tt.eventID, got, tt.want)
			}
		})
	}
}
