// Package store provides SQLite persistence for Nerva events.
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/isaacwallace123/cortex/services/nerva/internal/domain"
)

// SQLiteStore persists Nerva events to a local SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open nerva db at %s: %w", path, err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("nerva db migrate: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id         TEXT PRIMARY KEY,
			subsystem  TEXT NOT NULL,
			event_name TEXT NOT NULL,
			session_id TEXT NOT NULL DEFAULT '',
			payload    TEXT NOT NULL DEFAULT '{}',
			created_at INTEGER NOT NULL
		)
	`)
	return err
}

func (s *SQLiteStore) Save(_ context.Context, event domain.Event) error {
	_, err := s.db.Exec(
		`INSERT INTO events (id, subsystem, event_name, session_id, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.Subsystem,
		event.Name,
		event.SessionID,
		event.Payload,
		event.CreatedAt.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("nerva save event: %w", err)
	}
	return nil
}
