package client

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxSessionHistoryEntries = 200

// SessionHistoryEntry tracks a recently used chat session for the TUI.
type SessionHistoryEntry struct {
	SessionID string `json:"session_id"`
	Title     string `json:"title,omitempty"`
	LastUsed  int64  `json:"last_used"`
}

func sessionHistoryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cortex", "sessions.json"), nil
}

// LoadSessionHistory reads ~/.cortex/sessions.json.
// Returns an empty slice when the file does not exist.
func LoadSessionHistory() ([]SessionHistoryEntry, error) {
	path, err := sessionHistoryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []SessionHistoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []SessionHistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastUsed > entries[j].LastUsed
	})
	return entries, nil
}

// SaveSessionHistory writes ~/.cortex/sessions.json.
func SaveSessionHistory(entries []SessionHistoryEntry) error {
	path, err := sessionHistoryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// RecordSession upserts a session in local history and bumps its last-used time.
func RecordSession(sessionID, title string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	title = strings.TrimSpace(title)

	entries, err := LoadSessionHistory()
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	found := false
	for i := range entries {
		if entries[i].SessionID == sessionID {
			entries[i].LastUsed = now
			if title != "" {
				entries[i].Title = title
			}
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, SessionHistoryEntry{
			SessionID: sessionID,
			Title:     title,
			LastUsed:  now,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].LastUsed > entries[j].LastUsed
	})
	if len(entries) > maxSessionHistoryEntries {
		entries = entries[:maxSessionHistoryEntries]
	}
	return SaveSessionHistory(entries)
}

// DeleteSessionFromHistory removes a session from local history.
func DeleteSessionFromHistory(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	entries, err := LoadSessionHistory()
	if err != nil {
		return err
	}
	filtered := make([]SessionHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.SessionID == sessionID {
			continue
		}
		filtered = append(filtered, entry)
	}
	return SaveSessionHistory(filtered)
}
