package caldav

// Attendee represents a calendar event attendee.
type Attendee struct {
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
	Role   string `json:"role,omitempty"`
	Status string `json:"status,omitempty"`
}
