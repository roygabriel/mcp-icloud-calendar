package config

import (
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	ICloudEmail      string
	ICloudPassword   string
	ICloudCalendarID string // Optional default calendar ID
	LogLevel         string
	MaxConnsPerHost  int
	ToolTimeout      time.Duration
	MaxRetries       int
	RetryBaseDelay   time.Duration
	HealthPort       string
	RateLimitRPS     float64
	RateLimitBurst   int
	TLSCertFile      string
	TLSKeyFile       string
	TLSCAFile        string
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	email := os.Getenv("ICLOUD_EMAIL")
	password, err := loadCredential("ICLOUD_PASSWORD")
	if err != nil {
		return nil, err
	}
	calendarID := os.Getenv("ICLOUD_CALENDAR_ID")
	logLevel := os.Getenv("LOG_LEVEL")

	// Validate required fields
	if email == "" {
		return nil, fmt.Errorf("ICLOUD_EMAIL environment variable is required")
	}

	if password == "" {
		return nil, fmt.Errorf("ICLOUD_PASSWORD environment variable is required (use app-specific password from appleid.apple.com)")
	}

	if logLevel == "" {
		logLevel = "INFO"
	}

	maxConns, err := getIntEnv("MAX_CONNS_PER_HOST", 10)
	if err != nil {
		return nil, err
	}

	toolTimeout, err := getDurationEnv("TOOL_TIMEOUT", 25*time.Second)
	if err != nil {
		return nil, err
	}

	maxRetries, err := getIntEnv("MAX_RETRIES", 3)
	if err != nil {
		return nil, err
	}

	retryBaseDelay, err := getDurationEnv("RETRY_BASE_DELAY", 1*time.Second)
	if err != nil {
		return nil, err
	}

	healthPort := os.Getenv("HEALTH_PORT")

	rateLimitRPS, err := getFloatEnv("RATE_LIMIT_RPS", 10)
	if err != nil {
		return nil, err
	}

	rateLimitBurst, err := getIntEnv("RATE_LIMIT_BURST", 20)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ICloudEmail:      email,
		ICloudPassword:   password,
		ICloudCalendarID: calendarID,
		LogLevel:         logLevel,
		MaxConnsPerHost:  maxConns,
		ToolTimeout:      toolTimeout,
		MaxRetries:       maxRetries,
		RetryBaseDelay:   retryBaseDelay,
		HealthPort:       healthPort,
		RateLimitRPS:     rateLimitRPS,
		RateLimitBurst:   rateLimitBurst,
		TLSCertFile:      os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:       os.Getenv("TLS_KEY_FILE"),
		TLSCAFile:        os.Getenv("TLS_CA_FILE"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that config values are within acceptable ranges.
func (c *Config) Validate() error {
	if _, err := mail.ParseAddress(c.ICloudEmail); err != nil {
		return fmt.Errorf("invalid ICLOUD_EMAIL format: %w", err)
	}
	if len(c.ICloudPassword) < 4 {
		return fmt.Errorf("ICLOUD_PASSWORD is too short (minimum 4 characters)")
	}
	if c.ICloudCalendarID != "" && !strings.HasPrefix(c.ICloudCalendarID, "/") {
		return fmt.Errorf("ICLOUD_CALENDAR_ID must start with '/'")
	}
	if c.MaxConnsPerHost < 1 || c.MaxConnsPerHost > 100 {
		return fmt.Errorf("MAX_CONNS_PER_HOST must be between 1 and 100")
	}
	if c.ToolTimeout < 1*time.Second || c.ToolTimeout > 5*time.Minute {
		return fmt.Errorf("TOOL_TIMEOUT must be between 1s and 5m")
	}
	if c.MaxRetries < 0 || c.MaxRetries > 10 {
		return fmt.Errorf("MAX_RETRIES must be between 0 and 10")
	}
	if c.RetryBaseDelay < 100*time.Millisecond || c.RetryBaseDelay > 30*time.Second {
		return fmt.Errorf("RETRY_BASE_DELAY must be between 100ms and 30s")
	}
	return nil
}

// loadCredential reads a credential from an env var. If the value starts with
// "file://", the credential is read from the referenced file (for Docker/K8s secrets).
func loadCredential(envVar string) (string, error) {
	val := os.Getenv(envVar)
	if strings.HasPrefix(val, "file://") {
		path := strings.TrimPrefix(val, "file://")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read %s from file %s: %w", envVar, path, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return val, nil
}

func getIntEnv(key string, defaultVal int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}

func getFloatEnv(key string, defaultVal float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return f, nil
}

func getDurationEnv(key string, defaultVal time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return d, nil
}
