package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/isaacwallace123/cortex/services/policy/internal/domain"
)

// SQLiteStore implements ApprovalStore using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000&_foreign_keys=on")
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
		CREATE TABLE IF NOT EXISTS approvals (
			approval_id  TEXT    PRIMARY KEY,
			session_id   TEXT    NOT NULL DEFAULT '',
			plan_id      TEXT    NOT NULL DEFAULT '',
			step_index   INTEGER NOT NULL DEFAULT 0,
			description  TEXT    NOT NULL DEFAULT '',
			executor     TEXT    NOT NULL DEFAULT '',
			command      TEXT    NOT NULL DEFAULT '',
			reason       TEXT    NOT NULL DEFAULT '',
			user_id      TEXT    NOT NULL DEFAULT '',
			status       TEXT    NOT NULL DEFAULT 'pending',
			created_at   INTEGER NOT NULL,
			resolved_at  INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_approvals_user   ON approvals(user_id, status);
		CREATE INDEX IF NOT EXISTS idx_approvals_status ON approvals(status, created_at);
	`)
	return err
}

func (s *SQLiteStore) Create(ctx context.Context, a domain.Approval) (domain.Approval, error) {
	if a.ApprovalID == "" {
		a.ApprovalID = uuid.NewString()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	a.Status = domain.ApprovalPending

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO approvals
		 (approval_id, session_id, plan_id, step_index, description, executor, command, reason, user_id, status, created_at, resolved_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)`,
		a.ApprovalID, a.SessionID, a.PlanID, a.StepIndex, a.Description, a.Executor, a.Command, a.Reason, a.UserID, string(a.Status), a.CreatedAt.Unix(),
	)
	if err != nil {
		return domain.Approval{}, fmt.Errorf("insert approval: %w", err)
	}
	return a, nil
}

func (s *SQLiteStore) Resolve(ctx context.Context, approvalID, _ string, status domain.ApprovalStatus) error {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx,
		`UPDATE approvals SET status = ?, resolved_at = ? WHERE approval_id = ? AND status = 'pending'`,
		string(status), now, approvalID,
	)
	if err != nil {
		return fmt.Errorf("resolve approval: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("approval %s not found or already resolved", approvalID)
	}
	return nil
}

func (s *SQLiteStore) Get(ctx context.Context, approvalID string) (domain.Approval, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT approval_id, session_id, plan_id, step_index, description, executor, command, reason, user_id, status, created_at, resolved_at
		 FROM approvals WHERE approval_id = ?`, approvalID)
	return scanApproval(row)
}

func (s *SQLiteStore) ListPending(ctx context.Context, userID string) ([]domain.Approval, error) {
	query := `SELECT approval_id, session_id, plan_id, step_index, description, executor, command, reason, user_id, status, created_at, resolved_at
	          FROM approvals WHERE status = 'pending'`
	args := []any{}
	if userID != "" {
		query += ` AND user_id = ?`
		args = append(args, userID)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list pending approvals: %w", err)
	}
	defer rows.Close()

	var out []domain.Approval
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanApproval(s scanner) (domain.Approval, error) {
	var a domain.Approval
	var createdAt, resolvedAt int64
	var status string
	err := s.Scan(
		&a.ApprovalID, &a.SessionID, &a.PlanID, &a.StepIndex,
		&a.Description, &a.Executor, &a.Command, &a.Reason,
		&a.UserID, &status, &createdAt, &resolvedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Approval{}, fmt.Errorf("approval not found")
	}
	if err != nil {
		return domain.Approval{}, fmt.Errorf("scan approval: %w", err)
	}
	a.Status = domain.ApprovalStatus(status)
	a.CreatedAt = time.Unix(createdAt, 0).UTC()
	if resolvedAt > 0 {
		t := time.Unix(resolvedAt, 0).UTC()
		a.ResolvedAt = &t
	}
	return a, nil
}
