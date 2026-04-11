// Package store provides a SQLite-backed Vault storage adapter using FTS5 for
// full-text search across knowledge entries.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/vault/internal/domain"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS entries (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    workspace_id TEXT NOT NULL DEFAULT '',
    content      TEXT NOT NULL,
    source       TEXT NOT NULL DEFAULT '',
    tags         TEXT NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
    content,
    content='entries',
    content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS entries_ai AFTER INSERT ON entries BEGIN
    INSERT INTO entries_fts(rowid, content) VALUES (new.rowid, new.content);
END;

CREATE TRIGGER IF NOT EXISTS entries_ad AFTER DELETE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, content) VALUES ('delete', old.rowid, old.content);
END;

CREATE TRIGGER IF NOT EXISTS entries_au AFTER UPDATE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, content) VALUES ('delete', old.rowid, old.content);
    INSERT INTO entries_fts(rowid, content) VALUES (new.rowid, new.content);
END;
`

// SQLiteStore persists Vault entries in SQLite with FTS5 full-text indexing.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open vault db at %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("vault schema migration: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Store(ctx context.Context, entry *domain.Entry) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	tags := strings.Join(entry.Tags, ",")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO entries (id, user_id, workspace_id, content, source, tags, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.UserID, entry.WorkspaceID, entry.Content, entry.Source, tags, entry.CreatedAt.UnixMilli(),
	)
	return err
}

func (s *SQLiteStore) Search(ctx context.Context, userID, workspaceID, query string, limit int) ([]*domain.Entry, error) {
	if limit <= 0 {
		limit = 5
	}

	// Build workspace filter: empty workspaceID = all workspaces for this user.
	workspaceClause := ""
	args := []any{userID, query, limit}
	if workspaceID != "" {
		workspaceClause = "AND e.workspace_id = ?"
		args = []any{userID, workspaceID, query, limit}
	}

	q := fmt.Sprintf(`
		SELECT e.id, e.user_id, e.workspace_id, e.content, e.source, e.tags, e.created_at
		FROM entries e
		JOIN entries_fts ON e.rowid = entries_fts.rowid
		WHERE e.user_id = ? %s
		  AND entries_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, workspaceClause)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*domain.Entry
	for rows.Next() {
		e := &domain.Entry{}
		var tagsStr string
		var createdAtMs int64
		if err := rows.Scan(&e.ID, &e.UserID, &e.WorkspaceID, &e.Content, &e.Source, &tagsStr, &createdAtMs); err != nil {
			continue
		}
		if tagsStr != "" {
			e.Tags = strings.Split(tagsStr, ",")
		}
		e.CreatedAt = time.UnixMilli(createdAtMs)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *SQLiteStore) Delete(ctx context.Context, entryID, userID string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM entries WHERE id = ? AND user_id = ?`, entryID, userID,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
