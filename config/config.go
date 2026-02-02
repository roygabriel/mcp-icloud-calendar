package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	ICloudEmail      string
	ICloudPassword   string
	ICloudCalendarID string // Optional default calendar ID
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	email := os.Getenv("ICLOUD_EMAIL")
	password := os.Getenv("ICLOUD_PASSWORD")
	calendarID := os.Getenv("ICLOUD_CALENDAR_ID")

	// Validate required fields
	if email == "" {
		return nil, fmt.Errorf("ICLOUD_EMAIL environment variable is required")
	}

	if password == "" {
		return nil, fmt.Errorf("ICLOUD_PASSWORD environment variable is required (use app-specific password from appleid.apple.com)")
	}

	return &Config{
		ICloudEmail:      email,
		ICloudPassword:   password,
		ICloudCalendarID: calendarID,
	}, nil
}
