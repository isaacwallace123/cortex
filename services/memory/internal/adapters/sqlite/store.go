package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/isaacwallace123/cortex/services/memory/internal/domain"
)

// Store implements StorePort using SQLite via modernc.org/sqlite (pure Go, no cgo).
// The database file is created automatically at the given path.
// Use ":memory:" for ephemeral in-process storage (tests).
type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite at %s: %w", path, err)
	}

	// Single writer; multiple readers are fine for our workload.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id           TEXT    PRIMARY KEY,
			session_id   TEXT    NOT NULL,
			user_id      TEXT    NOT NULL DEFAULT '',
			workspace_id TEXT    NOT NULL DEFAULT '',
			event_name   TEXT    NOT NULL,
			payload      TEXT    NOT NULL DEFAULT '',
			stored_at    INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS plan_summaries (
			plan_id      TEXT    PRIMARY KEY,
			session_id   TEXT    NOT NULL,
			user_id      TEXT    NOT NULL DEFAULT '',
			workspace_id TEXT    NOT NULL DEFAULT '',
			intent       TEXT    NOT NULL DEFAULT '',
			step_count   INTEGER NOT NULL DEFAULT 0,
			created_at   INTEGER NOT NULL
		);
	`)
	if err != nil {
		return err
	}
	// Additive migration: add columns for databases created before M36.
	for _, col := range []struct{ table, name, def string }{
		{"events", "user_id", "TEXT NOT NULL DEFAULT ''"},
		{"events", "workspace_id", "TEXT NOT NULL DEFAULT ''"},
		{"plan_summaries", "user_id", "TEXT NOT NULL DEFAULT ''"},
		{"plan_summaries", "workspace_id", "TEXT NOT NULL DEFAULT ''"},
	} {
		_, _ = s.db.Exec(`ALTER TABLE ` + col.table + ` ADD COLUMN ` + col.name + ` ` + col.def)
		// Ignore error — column already exists on fresh databases.
	}

	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id, stored_at);
		CREATE INDEX IF NOT EXISTS idx_events_user    ON events(user_id, stored_at);
		CREATE INDEX IF NOT EXISTS idx_plans_session ON plan_summaries(session_id, created_at);
		CREATE INDEX IF NOT EXISTS idx_plans_user    ON plan_summaries(user_id, created_at);
	`)
	if err != nil {
		return err
	}

	// Additive migration: expires_at column for session TTL (0 = never expires).
	_, _ = s.db.Exec(`ALTER TABLE events ADD COLUMN expires_at INTEGER NOT NULL DEFAULT 0`)
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_events_expires ON events(expires_at) WHERE expires_at > 0`)
	if err != nil {
		return err
	}

	return nil
}

// PruneExpired deletes all events whose expires_at timestamp is in the past.
// It returns the number of rows deleted. Call this on a recurring timer to
// enforce session TTLs without blocking normal event operations.
func (s *Store) PruneExpired(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM events WHERE expires_at > 0 AND expires_at < ?`,
		time.Now().Unix(),
	)
	if err != nil {
		return 0, fmt.Errorf("prune expired events: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *Store) StoreEvent(ctx context.Context, event domain.Event) (domain.Event, error) {
	storedAt := event.StoredAt.Unix()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (id, session_id, user_id, workspace_id, event_name, payload, stored_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.SessionID, event.UserID, event.WorkspaceID, event.Name, event.Payload, storedAt,
	)
	if err != nil {
		return domain.Event{}, fmt.Errorf("insert event: %w", err)
	}

	// If this is a plan creation event, upsert the plan_summaries projection.
	if event.Name == "axiom.plan.created" {
		if err := s.upsertPlanSummary(ctx, event); err != nil {
			// Non-fatal — projection failure doesn't lose the event itself.
			_ = err
		}
	}

	return event, nil
}

func (s *Store) upsertPlanSummary(ctx context.Context, event domain.Event) error {
	planID := jsonString(event.Payload, "plan_id")
	intent := jsonString(event.Payload, "intent")
	stepCount := jsonInt(event.Payload, "step_count")
	if planID == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO plan_summaries (plan_id, session_id, user_id, workspace_id, intent, step_count, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		planID, event.SessionID, event.UserID, event.WorkspaceID, intent, stepCount, event.StoredAt.Unix(),
	)
	return err
}

func (s *Store) GetSession(ctx context.Context, sessionID, userID string) ([]domain.Event, error) {
	query := `SELECT id, session_id, user_id, workspace_id, event_name, payload, stored_at
	          FROM events WHERE session_id = ?`
	args := []any{sessionID}
	if userID != "" {
		query += ` AND user_id = ?`
		args = append(args, userID)
	}
	query += ` ORDER BY stored_at ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		var e domain.Event
		var storedAt int64
		if err := rows.Scan(&e.ID, &e.SessionID, &e.UserID, &e.WorkspaceID, &e.Name, &e.Payload, &storedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.StoredAt = time.Unix(storedAt, 0).UTC()
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) ListPlans(ctx context.Context, sessionID, userID string) ([]domain.PlanSummary, error) {
	query := `SELECT plan_id, session_id, user_id, workspace_id, intent, step_count, created_at
	          FROM plan_summaries WHERE session_id = ?`
	args := []any{sessionID}
	if userID != "" {
		query += ` AND user_id = ?`
		args = append(args, userID)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query plans: %w", err)
	}
	defer rows.Close()

	var plans []domain.PlanSummary
	for rows.Next() {
		var p domain.PlanSummary
		var createdAt int64
		if err := rows.Scan(&p.PlanID, &p.SessionID, &p.UserID, &p.WorkspaceID, &p.Intent, &p.StepCount, &createdAt); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		p.CreatedAt = time.Unix(createdAt, 0).UTC()
		plans = append(plans, p)
	}
	return plans, rows.Err()
}
