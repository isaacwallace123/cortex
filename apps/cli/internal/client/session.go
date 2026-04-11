package client

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// SavedSession is what gets written to ~/.cortex/session.json.
type SavedSession struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expires_at"`
}

func sessionPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cortex", "session.json"), nil
}

// LoadSession reads the locally saved session token. Returns empty struct if none.
func LoadSession() (SavedSession, error) {
	path, err := sessionPath()
	if err != nil {
		return SavedSession{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return SavedSession{}, nil
	}
	if err != nil {
		return SavedSession{}, err
	}
	var s SavedSession
	return s, json.Unmarshal(data, &s)
}

// SaveSession persists the session token to ~/.cortex/session.json.
func SaveSession(s SavedSession) error {
	path, err := sessionPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ClearSession deletes the saved session token.
func ClearSession() error {
	path, err := sessionPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
