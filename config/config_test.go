package config

import (
	"testing"
	"time"
)

func setDefaults(t *testing.T) {
	t.Helper()
	t.Setenv("ICLOUD_EMAIL", "user@example.com")
	t.Setenv("ICLOUD_PASSWORD", "testpass1234")
	t.Setenv("ICLOUD_CALENDAR_ID", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("MAX_CONNS_PER_HOST", "")
	t.Setenv("TOOL_TIMEOUT", "")
	t.Setenv("MAX_RETRIES", "")
	t.Setenv("RETRY_BASE_DELAY", "")
}

func TestLoad_RequiredFields(t *testing.T) {
	t.Run("missing email", func(t *testing.T) {
		setDefaults(t)
		t.Setenv("ICLOUD_EMAIL", "")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for missing email")
		}
	})

	t.Run("missing password", func(t *testing.T) {
		setDefaults(t)
		t.Setenv("ICLOUD_PASSWORD", "")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for missing password")
		}
	})
}

func TestLoad_Defaults(t *testing.T) {
	setDefaults(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.LogLevel != "INFO" {
		t.Errorf("LogLevel = %q, want INFO", cfg.LogLevel)
	}
	if cfg.MaxConnsPerHost != 10 {
		t.Errorf("MaxConnsPerHost = %d, want 10", cfg.MaxConnsPerHost)
	}
	if cfg.ToolTimeout != 25*time.Second {
		t.Errorf("ToolTimeout = %v, want 25s", cfg.ToolTimeout)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.RetryBaseDelay != 1*time.Second {
		t.Errorf("RetryBaseDelay = %v, want 1s", cfg.RetryBaseDelay)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	setDefaults(t)
	t.Setenv("ICLOUD_CALENDAR_ID", "/cal/work")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("MAX_CONNS_PER_HOST", "20")
	t.Setenv("TOOL_TIMEOUT", "30s")
	t.Setenv("MAX_RETRIES", "5")
	t.Setenv("RETRY_BASE_DELAY", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ICloudCalendarID != "/cal/work" {
		t.Errorf("ICloudCalendarID = %q, want /cal/work", cfg.ICloudCalendarID)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("LogLevel = %q, want DEBUG", cfg.LogLevel)
	}
	if cfg.MaxConnsPerHost != 20 {
		t.Errorf("MaxConnsPerHost = %d, want 20", cfg.MaxConnsPerHost)
	}
	if cfg.ToolTimeout != 30*time.Second {
		t.Errorf("ToolTimeout = %v, want 30s", cfg.ToolTimeout)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
}

func TestLoad_InvalidValues(t *testing.T) {
	t.Run("invalid MAX_CONNS_PER_HOST", func(t *testing.T) {
		setDefaults(t)
		t.Setenv("MAX_CONNS_PER_HOST", "abc")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for invalid MAX_CONNS_PER_HOST")
		}
	})

	t.Run("invalid TOOL_TIMEOUT", func(t *testing.T) {
		setDefaults(t)
		t.Setenv("TOOL_TIMEOUT", "invalid")
		_, err := Load()
		if err == nil {
			t.Fatal("expected error for invalid TOOL_TIMEOUT")
		}
	})
}

func TestValidate_EmailFormat(t *testing.T) {
	setDefaults(t)
	t.Setenv("ICLOUD_EMAIL", "not-an-email")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid email format")
	}
}

func TestValidate_PasswordTooShort(t *testing.T) {
	setDefaults(t)
	t.Setenv("ICLOUD_PASSWORD", "ab")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestValidate_CalendarIDMustStartWithSlash(t *testing.T) {
	setDefaults(t)
	t.Setenv("ICLOUD_CALENDAR_ID", "cal/work")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for calendar ID without leading slash")
	}
}

func TestValidate_MaxConnsOutOfRange(t *testing.T) {
	setDefaults(t)
	t.Setenv("MAX_CONNS_PER_HOST", "0")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for MaxConnsPerHost = 0")
	}
}
