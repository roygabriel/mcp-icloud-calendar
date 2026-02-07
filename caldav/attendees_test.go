package caldav

import (
	"encoding/json"
	"testing"
)

func TestAttendee_JSON(t *testing.T) {
	a := Attendee{
		Email:  "user@example.com",
		Name:   "Test User",
		Role:   "REQ-PARTICIPANT",
		Status: "ACCEPTED",
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("failed to marshal attendee: %v", err)
	}

	var decoded Attendee
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal attendee: %v", err)
	}

	if decoded.Email != a.Email {
		t.Errorf("email = %q, want %q", decoded.Email, a.Email)
	}
	if decoded.Name != a.Name {
		t.Errorf("name = %q, want %q", decoded.Name, a.Name)
	}
}
