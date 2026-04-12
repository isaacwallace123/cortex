package client

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config holds persistent CLI settings written to ~/.config/cortex/config.json.
// Flags and environment variables always take precedence over this file.
type Config struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
}

func configPath() (string, error) {
	// Respect XDG_CONFIG_HOME if set, otherwise default to ~/.config.
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "cortex", "config.json"), nil
}

// LoadConfig reads ~/.config/cortex/config.json. Returns an empty Config if the
// file does not exist yet — callers should treat missing fields as "not set".
func LoadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	return c, json.Unmarshal(data, &c)
}

// SaveConfig writes cfg to ~/.config/cortex/config.json, creating directories
// as needed with restrictive permissions (0700 / 0600).
func SaveConfig(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
