package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Account represents a single iCloud account configuration.
type Account struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	CalendarID string `json:"calendarId,omitempty"`
}

// AccountsConfig holds multiple account configurations.
type AccountsConfig struct {
	Accounts []Account `json:"accounts"`
}

// LoadAccounts loads account configurations from ACCOUNTS_FILE.
// If ACCOUNTS_FILE is not set, falls back to the single account from env vars.
func LoadAccounts(cfg *Config) (map[string]Account, error) {
	accountsFile := os.Getenv("ACCOUNTS_FILE")
	if accountsFile == "" {
		// Single account mode: use main config
		return map[string]Account{
			"default": {
				Name:       "default",
				Email:      cfg.ICloudEmail,
				Password:   cfg.ICloudPassword,
				CalendarID: cfg.ICloudCalendarID,
			},
		}, nil
	}

	data, err := os.ReadFile(accountsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read accounts file %s: %w", accountsFile, err)
	}

	var ac AccountsConfig
	if err := json.Unmarshal(data, &ac); err != nil {
		return nil, fmt.Errorf("failed to parse accounts file: %w", err)
	}

	if len(ac.Accounts) == 0 {
		return nil, fmt.Errorf("accounts file contains no accounts")
	}

	accounts := make(map[string]Account, len(ac.Accounts))
	for _, a := range ac.Accounts {
		if a.Name == "" {
			return nil, fmt.Errorf("account is missing 'name' field")
		}
		if a.Email == "" {
			return nil, fmt.Errorf("account %q is missing 'email' field", a.Name)
		}
		if a.Password == "" {
			return nil, fmt.Errorf("account %q is missing 'password' field", a.Name)
		}
		accounts[a.Name] = a
	}

	return accounts, nil
}
