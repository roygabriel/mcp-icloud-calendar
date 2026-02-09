package config

import (
	"os"
	"path/filepath"
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
	t.Setenv("RATE_LIMIT_RPS", "")
	t.Setenv("RATE_LIMIT_BURST", "")
	t.Setenv("HEALTH_PORT", "")
	t.Setenv("TLS_CERT_FILE", "")
	t.Setenv("TLS_KEY_FILE", "")
	t.Setenv("TLS_CA_FILE", "")
	t.Setenv("ACCOUNTS_FILE", "")
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

func TestValidate_MaxConnsAboveRange(t *testing.T) {
	setDefaults(t)
	t.Setenv("MAX_CONNS_PER_HOST", "101")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for MaxConnsPerHost = 101")
	}
}

func TestValidate_ToolTimeoutTooLow(t *testing.T) {
	setDefaults(t)
	t.Setenv("TOOL_TIMEOUT", "100ms")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for ToolTimeout below 1s")
	}
}

func TestValidate_ToolTimeoutTooHigh(t *testing.T) {
	setDefaults(t)
	t.Setenv("TOOL_TIMEOUT", "10m")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for ToolTimeout above 5m")
	}
}

func TestValidate_MaxRetriesTooHigh(t *testing.T) {
	setDefaults(t)
	t.Setenv("MAX_RETRIES", "11")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for MaxRetries > 10")
	}
}

func TestValidate_MaxRetriesTooLow(t *testing.T) {
	setDefaults(t)
	t.Setenv("MAX_RETRIES", "-1")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for MaxRetries < 0")
	}
}

func TestValidate_RetryBaseDelayTooLow(t *testing.T) {
	setDefaults(t)
	t.Setenv("RETRY_BASE_DELAY", "10ms")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for RetryBaseDelay below 100ms")
	}
}

func TestValidate_RetryBaseDelayTooHigh(t *testing.T) {
	setDefaults(t)
	t.Setenv("RETRY_BASE_DELAY", "60s")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for RetryBaseDelay above 30s")
	}
}

func TestLoad_InvalidMaxRetries(t *testing.T) {
	setDefaults(t)
	t.Setenv("MAX_RETRIES", "xyz")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric MAX_RETRIES")
	}
}

func TestLoad_InvalidRetryBaseDelay(t *testing.T) {
	setDefaults(t)
	t.Setenv("RETRY_BASE_DELAY", "not-a-duration")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid RETRY_BASE_DELAY")
	}
}

func TestLoad_InvalidRateLimitRPS(t *testing.T) {
	setDefaults(t)
	t.Setenv("RATE_LIMIT_RPS", "abc")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric RATE_LIMIT_RPS")
	}
}

func TestLoad_InvalidRateLimitBurst(t *testing.T) {
	setDefaults(t)
	t.Setenv("RATE_LIMIT_BURST", "xyz")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric RATE_LIMIT_BURST")
	}
}

func TestLoad_CustomRateLimitValues(t *testing.T) {
	setDefaults(t)
	t.Setenv("RATE_LIMIT_RPS", "5.5")
	t.Setenv("RATE_LIMIT_BURST", "15")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RateLimitRPS != 5.5 {
		t.Errorf("RateLimitRPS = %v, want 5.5", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 15 {
		t.Errorf("RateLimitBurst = %d, want 15", cfg.RateLimitBurst)
	}
}

func TestLoad_HealthPort(t *testing.T) {
	setDefaults(t)
	t.Setenv("HEALTH_PORT", "8080")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HealthPort != "8080" {
		t.Errorf("HealthPort = %q, want 8080", cfg.HealthPort)
	}
}

func TestLoad_TLSConfig(t *testing.T) {
	setDefaults(t)
	t.Setenv("TLS_CERT_FILE", "/path/to/cert.pem")
	t.Setenv("TLS_KEY_FILE", "/path/to/key.pem")
	t.Setenv("TLS_CA_FILE", "/path/to/ca.pem")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TLSCertFile != "/path/to/cert.pem" {
		t.Errorf("TLSCertFile = %q", cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile != "/path/to/key.pem" {
		t.Errorf("TLSKeyFile = %q", cfg.TLSKeyFile)
	}
	if cfg.TLSCAFile != "/path/to/ca.pem" {
		t.Errorf("TLSCAFile = %q", cfg.TLSCAFile)
	}
}

func TestLoadCredential_FileProtocol(t *testing.T) {
	dir := t.TempDir()
	secretFile := filepath.Join(dir, "password.txt")
	if err := os.WriteFile(secretFile, []byte("  secret-pass  \n"), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	setDefaults(t)
	t.Setenv("ICLOUD_PASSWORD", "file://"+secretFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ICloudPassword != "secret-pass" {
		t.Errorf("password = %q, want 'secret-pass' (trimmed)", cfg.ICloudPassword)
	}
}

func TestLoadCredential_FileNotFound(t *testing.T) {
	setDefaults(t)
	t.Setenv("ICLOUD_PASSWORD", "file:///nonexistent/secret")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for file not found")
	}
}

func TestLoad_DefaultRateLimitValues(t *testing.T) {
	setDefaults(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RateLimitRPS != 10 {
		t.Errorf("RateLimitRPS = %v, want 10", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 20 {
		t.Errorf("RateLimitBurst = %d, want 20", cfg.RateLimitBurst)
	}
}
