package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAccounts_SingleAccountMode(t *testing.T) {
	cfg := &Config{
		ICloudEmail:      "user@example.com",
		ICloudPassword:   "pass1234",
		ICloudCalendarID: "/cal/default",
	}
	t.Setenv("ACCOUNTS_FILE", "")

	accounts, err := LoadAccounts(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}

	acct, ok := accounts["default"]
	if !ok {
		t.Fatal("expected 'default' account")
	}
	if acct.Email != "user@example.com" {
		t.Errorf("email = %q, want user@example.com", acct.Email)
	}
	if acct.Password != "pass1234" {
		t.Errorf("password = %q, want pass1234", acct.Password)
	}
	if acct.CalendarID != "/cal/default" {
		t.Errorf("calendarID = %q, want /cal/default", acct.CalendarID)
	}
}

func TestLoadAccounts_MultiAccountMode(t *testing.T) {
	dir := t.TempDir()
	accountsFile := filepath.Join(dir, "accounts.json")

	content := `{
		"accounts": [
			{"name": "work", "email": "work@example.com", "password": "workpass", "calendarId": "/cal/work"},
			{"name": "personal", "email": "personal@example.com", "password": "perspass"}
		]
	}`
	if err := os.WriteFile(accountsFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write accounts file: %v", err)
	}

	t.Setenv("ACCOUNTS_FILE", accountsFile)

	accounts, err := LoadAccounts(&Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	work := accounts["work"]
	if work.Email != "work@example.com" {
		t.Errorf("work email = %q, want work@example.com", work.Email)
	}
	if work.CalendarID != "/cal/work" {
		t.Errorf("work calendarId = %q, want /cal/work", work.CalendarID)
	}

	personal := accounts["personal"]
	if personal.Email != "personal@example.com" {
		t.Errorf("personal email = %q", personal.Email)
	}
}

func TestLoadAccounts_FileNotFound(t *testing.T) {
	t.Setenv("ACCOUNTS_FILE", "/nonexistent/accounts.json")

	_, err := LoadAccounts(&Config{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadAccounts_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	accountsFile := filepath.Join(dir, "accounts.json")
	if err := os.WriteFile(accountsFile, []byte("not json"), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	t.Setenv("ACCOUNTS_FILE", accountsFile)

	_, err := LoadAccounts(&Config{})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadAccounts_EmptyAccounts(t *testing.T) {
	dir := t.TempDir()
	accountsFile := filepath.Join(dir, "accounts.json")
	if err := os.WriteFile(accountsFile, []byte(`{"accounts": []}`), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	t.Setenv("ACCOUNTS_FILE", accountsFile)

	_, err := LoadAccounts(&Config{})
	if err == nil {
		t.Fatal("expected error for empty accounts")
	}
}

func TestLoadAccounts_MissingName(t *testing.T) {
	dir := t.TempDir()
	accountsFile := filepath.Join(dir, "accounts.json")
	content := `{"accounts": [{"email": "test@example.com", "password": "pass"}]}`
	if err := os.WriteFile(accountsFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	t.Setenv("ACCOUNTS_FILE", accountsFile)

	_, err := LoadAccounts(&Config{})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoadAccounts_MissingEmail(t *testing.T) {
	dir := t.TempDir()
	accountsFile := filepath.Join(dir, "accounts.json")
	content := `{"accounts": [{"name": "work", "password": "pass"}]}`
	if err := os.WriteFile(accountsFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	t.Setenv("ACCOUNTS_FILE", accountsFile)

	_, err := LoadAccounts(&Config{})
	if err == nil {
		t.Fatal("expected error for missing email")
	}
}

func TestLoadAccounts_MissingPassword(t *testing.T) {
	dir := t.TempDir()
	accountsFile := filepath.Join(dir, "accounts.json")
	content := `{"accounts": [{"name": "work", "email": "work@example.com"}]}`
	if err := os.WriteFile(accountsFile, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	t.Setenv("ACCOUNTS_FILE", accountsFile)

	_, err := LoadAccounts(&Config{})
	if err == nil {
		t.Fatal("expected error for missing password")
	}
}
