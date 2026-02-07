package tools

import (
	"testing"

	"github.com/rgabriel/mcp-icloud-calendar/caldav"
)

// testAccounts creates an AccountClients with a single "default" account for testing.
func testAccounts(mock caldav.CalendarService, defaultCalendar string) *AccountClients {
	return NewAccountClients(
		map[string]caldav.CalendarService{"default": mock},
		map[string]string{"default": defaultCalendar},
	)
}

// testMultiAccounts creates an AccountClients with multiple accounts for testing.
func testMultiAccounts(mocks map[string]caldav.CalendarService, defaults map[string]string) *AccountClients {
	return NewAccountClients(mocks, defaults)
}

func TestAccountClients_ResolveDefault(t *testing.T) {
	mock := &caldav.MockClient{}
	ac := testAccounts(mock, "/cal/default")

	client, cal, err := ac.Resolve("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client != mock {
		t.Error("expected default mock client")
	}
	if cal != "/cal/default" {
		t.Errorf("default calendar = %q, want /cal/default", cal)
	}
}

func TestAccountClients_ResolveByName(t *testing.T) {
	mock1 := &caldav.MockClient{CreatedEventID: "mock1"}
	mock2 := &caldav.MockClient{CreatedEventID: "mock2"}
	ac := testMultiAccounts(
		map[string]caldav.CalendarService{"work": mock1, "personal": mock2},
		map[string]string{"work": "/cal/work", "personal": "/cal/personal"},
	)

	client, cal, err := ac.Resolve("personal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client != mock2 {
		t.Error("expected personal mock client")
	}
	if cal != "/cal/personal" {
		t.Errorf("calendar = %q, want /cal/personal", cal)
	}
}

func TestAccountClients_ResolveUnknown(t *testing.T) {
	mock := &caldav.MockClient{}
	ac := testAccounts(mock, "/cal/default")

	_, _, err := ac.Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown account")
	}
}

func TestAccountClients_AccountNames(t *testing.T) {
	mock := &caldav.MockClient{}
	ac := testMultiAccounts(
		map[string]caldav.CalendarService{"work": mock, "personal": mock},
		map[string]string{"work": "/cal/work", "personal": "/cal/personal"},
	)

	names := ac.AccountNames()
	if len(names) != 2 {
		t.Errorf("AccountNames() returned %d names, want 2", len(names))
	}
}
