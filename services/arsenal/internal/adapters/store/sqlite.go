// Package store provides a SQLite-backed implementation of ports.ToolStore.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/isaacwallace123/cortex/services/arsenal/internal/domain"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS tools (
    name         TEXT PRIMARY KEY,
    description  TEXT NOT NULL DEFAULT '',
    executor     TEXT NOT NULL DEFAULT '',
    example      TEXT NOT NULL DEFAULT '',
    safety_level TEXT NOT NULL DEFAULT 'safe',
    tags         TEXT NOT NULL DEFAULT '',
    enabled      INTEGER NOT NULL DEFAULT 1,
    version      TEXT NOT NULL DEFAULT '1.0.0'
);
`

// SQLiteStore persists Arsenal tool definitions in SQLite.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open arsenal db at %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("arsenal schema migration: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Upsert(ctx context.Context, t *domain.Tool) error {
	tags := strings.Join(t.Tags, ",")
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tools (name, description, executor, example, safety_level, tags, enabled, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			description  = excluded.description,
			executor     = excluded.executor,
			example      = excluded.example,
			safety_level = excluded.safety_level,
			tags         = excluded.tags,
			enabled      = excluded.enabled,
			version      = excluded.version`,
		t.Name, t.Description, t.Executor, t.Example,
		string(t.SafetyLevel), tags, enabled, t.Version,
	)
	return err
}

func (s *SQLiteStore) Get(ctx context.Context, name string) (*domain.Tool, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT name, description, executor, example, safety_level, tags, enabled, version
		 FROM tools WHERE name = ?`, name)
	return scanTool(row)
}

func (s *SQLiteStore) List(ctx context.Context, executor, tag string) ([]*domain.Tool, error) {
	q := `SELECT name, description, executor, example, safety_level, tags, enabled, version FROM tools WHERE enabled = 1`
	var args []any
	if executor != "" {
		q += " AND executor = ?"
		args = append(args, executor)
	}
	rows, err := s.db.QueryContext(ctx, q+" ORDER BY name", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []*domain.Tool
	for rows.Next() {
		t, err := scanToolRow(rows)
		if err != nil {
			continue
		}
		if tag != "" {
			found := false
			for _, tg := range t.Tags {
				if tg == tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		tools = append(tools, t)
	}
	return tools, rows.Err()
}

func (s *SQLiteStore) Delete(ctx context.Context, name string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tools WHERE name = ?`, name)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// scanner is a common interface satisfied by both *sql.Row and *sql.Rows,
// allowing scan helpers to work with both single and multi-row queries.
type scanner interface {
	Scan(dest ...any) error
}

func scanTool(row *sql.Row) (*domain.Tool, error) {
	var (
		t       domain.Tool
		tagsStr string
		enabled int
		safety  string
	)
	err := row.Scan(&t.Name, &t.Description, &t.Executor, &t.Example,
		&safety, &tagsStr, &enabled, &t.Version)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.SafetyLevel = domain.SafetyLevel(safety)
	t.Enabled = enabled != 0
	if tagsStr != "" {
		t.Tags = strings.Split(tagsStr, ",")
	}
	return &t, nil
}

func scanToolRow(rows *sql.Rows) (*domain.Tool, error) {
	var (
		t       domain.Tool
		tagsStr string
		enabled int
		safety  string
	)
	if err := rows.Scan(&t.Name, &t.Description, &t.Executor, &t.Example,
		&safety, &tagsStr, &enabled, &t.Version); err != nil {
		return nil, err
	}
	t.SafetyLevel = domain.SafetyLevel(safety)
	t.Enabled = enabled != 0
	if tagsStr != "" {
		t.Tags = strings.Split(tagsStr, ",")
	}
	return &t, nil
}

