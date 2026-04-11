package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/isaacwallace123/cortex/services/genesis/internal/domain"
)

// SQLiteStore implements ProposalStore using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
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
		CREATE TABLE IF NOT EXISTS proposals (
			proposal_id TEXT    PRIMARY KEY,
			description TEXT    NOT NULL DEFAULT '',
			tool_json   TEXT    NOT NULL DEFAULT 'null',
			registered  INTEGER NOT NULL DEFAULT 0,
			error       TEXT    NOT NULL DEFAULT '',
			created_at  INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_proposals_created ON proposals(created_at DESC);
	`)
	return err
}

func (s *SQLiteStore) Save(ctx context.Context, p domain.Proposal) (domain.Proposal, error) {
	if p.ProposalID == "" {
		p.ProposalID = uuid.NewString()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	toolJSON, err := json.Marshal(p.Tool)
	if err != nil {
		return domain.Proposal{}, fmt.Errorf("marshal tool: %w", err)
	}
	registered := 0
	if p.Registered {
		registered = 1
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO proposals (proposal_id, description, tool_json, registered, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.ProposalID, p.Description, string(toolJSON), registered, p.Error, p.CreatedAt.Unix(),
	)
	if err != nil {
		return domain.Proposal{}, fmt.Errorf("insert proposal: %w", err)
	}
	return p, nil
}

func (s *SQLiteStore) List(ctx context.Context) ([]domain.Proposal, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT proposal_id, description, tool_json, registered, error, created_at
		 FROM proposals ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query proposals: %w", err)
	}
	defer rows.Close()

	var proposals []domain.Proposal
	for rows.Next() {
		var p domain.Proposal
		var toolJSON string
		var registered int
		var createdAt int64
		if err := rows.Scan(&p.ProposalID, &p.Description, &toolJSON, &registered, &p.Error, &createdAt); err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		if err := json.Unmarshal([]byte(toolJSON), &p.Tool); err != nil {
			return nil, fmt.Errorf("unmarshal tool: %w", err)
		}
		p.Registered = registered == 1
		p.CreatedAt = time.Unix(createdAt, 0).UTC()
		proposals = append(proposals, p)
	}
	return proposals, rows.Err()
}
