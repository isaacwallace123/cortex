package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/isaacwallace123/cortex/services/workspace/internal/domain"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite at %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS workspaces (
			workspace_id TEXT    PRIMARY KEY,
			user_id      TEXT    NOT NULL,
			name         TEXT    NOT NULL,
			description  TEXT    NOT NULL DEFAULT '',
			root_path    TEXT    NOT NULL DEFAULT '',
			created_at   INTEGER NOT NULL,
			updated_at   INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_workspaces_user ON workspaces(user_id, created_at);
	`)
	return err
}

func (s *SQLiteStore) Create(ctx context.Context, w domain.Workspace) (domain.Workspace, error) {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workspaces (workspace_id, user_id, name, description, root_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		w.WorkspaceID, w.UserID, w.Name, w.Description, w.RootPath,
		w.CreatedAt.Unix(), w.UpdatedAt.Unix(),
	)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("insert workspace: %w", err)
	}
	return w, nil
}

func (s *SQLiteStore) Get(ctx context.Context, workspaceID, userID string) (domain.Workspace, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT workspace_id, user_id, name, description, root_path, created_at, updated_at
		 FROM workspaces WHERE workspace_id = ? AND user_id = ?`,
		workspaceID, userID,
	)
	return scanWorkspace(row)
}

func (s *SQLiteStore) List(ctx context.Context, userID string) ([]domain.Workspace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT workspace_id, user_id, name, description, root_path, created_at, updated_at
		 FROM workspaces WHERE user_id = ? ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query workspaces: %w", err)
	}
	defer rows.Close()

	var out []domain.Workspace
	for rows.Next() {
		w, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) Update(ctx context.Context, w domain.Workspace) (domain.Workspace, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE workspaces SET name=?, description=?, root_path=?, updated_at=?
		 WHERE workspace_id=? AND user_id=?`,
		w.Name, w.Description, w.RootPath, w.UpdatedAt.Unix(),
		w.WorkspaceID, w.UserID,
	)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("update workspace: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.Workspace{}, fmt.Errorf("workspace not found")
	}
	return s.Get(ctx, w.WorkspaceID, w.UserID)
}

func (s *SQLiteStore) Delete(ctx context.Context, workspaceID, userID string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM workspaces WHERE workspace_id=? AND user_id=?`,
		workspaceID, userID,
	)
	if err != nil {
		return false, fmt.Errorf("delete workspace: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanWorkspace(row scanner) (domain.Workspace, error) {
	var w domain.Workspace
	var createdAt, updatedAt int64
	if err := row.Scan(&w.WorkspaceID, &w.UserID, &w.Name, &w.Description, &w.RootPath, &createdAt, &updatedAt); err != nil {
		return domain.Workspace{}, fmt.Errorf("scan workspace: %w", err)
	}
	w.CreatedAt = time.Unix(createdAt, 0).UTC()
	w.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return w, nil
}

