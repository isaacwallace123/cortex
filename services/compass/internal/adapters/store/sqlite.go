// Package store provides a SQLite-backed implementation of ports.ProjectStore.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/compass/internal/domain"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS projects (
    project_id  TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'planning',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS milestones (
    milestone_id TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',
    ord          INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tasks (
    task_id              TEXT PRIMARY KEY,
    milestone_id         TEXT NOT NULL REFERENCES milestones(milestone_id) ON DELETE CASCADE,
    name                 TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL DEFAULT 'todo',
    acceptance_criteria  TEXT NOT NULL DEFAULT '',
    created_at           INTEGER NOT NULL,
    completed_at         INTEGER
);

CREATE TABLE IF NOT EXISTS decisions (
    decision_id  TEXT PRIMARY KEY,
    project_id   TEXT NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    rationale    TEXT NOT NULL DEFAULT '',
    alternatives TEXT NOT NULL DEFAULT '',
    decided_at   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_milestones_project ON milestones(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_milestone    ON tasks(milestone_id);
CREATE INDEX IF NOT EXISTS idx_decisions_project  ON decisions(project_id);
`

// SQLiteStore persists Compass data in SQLite.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open compass db at %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("compass schema migration: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) CreateProject(ctx context.Context, p *domain.Project) error {
	if p.ProjectID == "" {
		p.ProjectID = uuid.NewString()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (project_id, name, description, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.ProjectID, p.Name, p.Description, string(p.Status),
		now.UnixMilli(), now.UnixMilli(),
	)
	return err
}

func (s *SQLiteStore) GetProject(ctx context.Context, projectID string) (*domain.Project, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT project_id, name, description, status, created_at, updated_at FROM projects WHERE project_id = ?`,
		projectID,
	)
	return scanProject(row)
}

func (s *SQLiteStore) ListProjects(ctx context.Context, statusFilter domain.ProjectStatus) ([]*domain.Project, error) {
	q := `SELECT project_id, name, description, status, created_at, updated_at FROM projects`
	var args []any
	if statusFilter != "" {
		q += " WHERE status = ?"
		args = append(args, string(statusFilter))
	}
	q += " ORDER BY created_at DESC"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Project
	for rows.Next() {
		p, err := scanProjectRow(rows)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateProjectStatus(ctx context.Context, projectID string, status domain.ProjectStatus) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE projects SET status = ?, updated_at = ? WHERE project_id = ?`,
		string(status), time.Now().UnixMilli(), projectID,
	)
	return err
}

func (s *SQLiteStore) CreateMilestone(ctx context.Context, m *domain.Milestone) error {
	if m.MilestoneID == "" {
		m.MilestoneID = uuid.NewString()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO milestones (milestone_id, project_id, name, description, status, ord)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		m.MilestoneID, m.ProjectID, m.Name, m.Description, string(m.Status), m.Order,
	)
	return err
}

func (s *SQLiteStore) ListMilestones(ctx context.Context, projectID string) ([]*domain.Milestone, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT milestone_id, project_id, name, description, status, ord FROM milestones
		 WHERE project_id = ? ORDER BY ord ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Milestone
	for rows.Next() {
		m := &domain.Milestone{}
		var status string
		if err := rows.Scan(&m.MilestoneID, &m.ProjectID, &m.Name, &m.Description, &status, &m.Order); err != nil {
			continue
		}
		m.Status = domain.MilestoneStatus(status)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateMilestoneStatus(ctx context.Context, milestoneID string, status domain.MilestoneStatus) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE milestones SET status = ? WHERE milestone_id = ?`,
		string(status), milestoneID,
	)
	return err
}

func (s *SQLiteStore) CreateTask(ctx context.Context, t *domain.Task) error {
	if t.TaskID == "" {
		t.TaskID = uuid.NewString()
	}
	t.CreatedAt = time.Now()
	criteria := strings.Join(t.AcceptanceCriteria, "\n")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (task_id, milestone_id, name, description, status, acceptance_criteria, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.TaskID, t.MilestoneID, t.Name, t.Description, string(t.Status), criteria, t.CreatedAt.UnixMilli(),
	)
	return err
}

func (s *SQLiteStore) ListTasks(ctx context.Context, milestoneID string) ([]*domain.Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT task_id, milestone_id, name, description, status, acceptance_criteria, created_at, completed_at
		 FROM tasks WHERE milestone_id = ? ORDER BY created_at ASC`,
		milestoneID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Task
	for rows.Next() {
		t := &domain.Task{}
		var status, criteria string
		var createdMs int64
		var completedMs *int64
		if err := rows.Scan(&t.TaskID, &t.MilestoneID, &t.Name, &t.Description,
			&status, &criteria, &createdMs, &completedMs); err != nil {
			continue
		}
		t.Status = domain.TaskStatus(status)
		t.CreatedAt = time.UnixMilli(createdMs)
		if completedMs != nil {
			ts := time.UnixMilli(*completedMs)
			t.CompletedAt = &ts
		}
		if criteria != "" {
			t.AcceptanceCriteria = strings.Split(criteria, "\n")
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateTaskStatus(ctx context.Context, taskID string, status domain.TaskStatus) error {
	var completedAt *int64
	if status == domain.TaskDone {
		ms := time.Now().UnixMilli()
		completedAt = &ms
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, completed_at = ? WHERE task_id = ?`,
		string(status), completedAt, taskID,
	)
	return err
}

func (s *SQLiteStore) AddDecision(ctx context.Context, d *domain.Decision) error {
	if d.DecisionID == "" {
		d.DecisionID = uuid.NewString()
	}
	d.DecidedAt = time.Now()
	alts := strings.Join(d.Alternatives, "\n")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO decisions (decision_id, project_id, title, rationale, alternatives, decided_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		d.DecisionID, d.ProjectID, d.Title, d.Rationale, alts, d.DecidedAt.UnixMilli(),
	)
	return err
}

func (s *SQLiteStore) ListDecisions(ctx context.Context, projectID string) ([]*domain.Decision, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT decision_id, project_id, title, rationale, alternatives, decided_at
		 FROM decisions WHERE project_id = ? ORDER BY decided_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Decision
	for rows.Next() {
		d := &domain.Decision{}
		var alts string
		var decidedMs int64
		if err := rows.Scan(&d.DecisionID, &d.ProjectID, &d.Title, &d.Rationale, &alts, &decidedMs); err != nil {
			continue
		}
		d.DecidedAt = time.UnixMilli(decidedMs)
		if alts != "" {
			d.Alternatives = strings.Split(alts, "\n")
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanProject(row *sql.Row) (*domain.Project, error) {
	p := &domain.Project{}
	var status string
	var createdMs, updatedMs int64
	if err := row.Scan(&p.ProjectID, &p.Name, &p.Description, &status, &createdMs, &updatedMs); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.Status = domain.ProjectStatus(status)
	p.CreatedAt = time.UnixMilli(createdMs)
	p.UpdatedAt = time.UnixMilli(updatedMs)
	return p, nil
}

func scanProjectRow(rows *sql.Rows) (*domain.Project, error) {
	p := &domain.Project{}
	var status string
	var createdMs, updatedMs int64
	if err := rows.Scan(&p.ProjectID, &p.Name, &p.Description, &status, &createdMs, &updatedMs); err != nil {
		return nil, err
	}
	p.Status = domain.ProjectStatus(status)
	p.CreatedAt = time.UnixMilli(createdMs)
	p.UpdatedAt = time.UnixMilli(updatedMs)
	return p, nil
}

