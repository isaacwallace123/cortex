package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/isaacwallace123/cortex/services/catalyst/internal/domain"
)

// SQLiteStore implements SkillStore using SQLite.
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
		CREATE TABLE IF NOT EXISTS skills (
			name        TEXT    PRIMARY KEY,
			description TEXT    NOT NULL DEFAULT '',
			steps_json  TEXT    NOT NULL DEFAULT '[]',
			tags_json   TEXT    NOT NULL DEFAULT '[]',
			version     TEXT    NOT NULL DEFAULT '1.0.0',
			created_at  INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_skills_created ON skills(created_at);
	`)
	return err
}

func (s *SQLiteStore) Create(ctx context.Context, skill domain.Skill) (domain.Skill, error) {
	if skill.Name == "" {
		skill.Name = uuid.NewString()
	}
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = time.Now().UTC()
	}
	stepsJSON, err := json.Marshal(skill.Steps)
	if err != nil {
		return domain.Skill{}, fmt.Errorf("marshal steps: %w", err)
	}
	tagsJSON, err := json.Marshal(skill.Tags)
	if err != nil {
		return domain.Skill{}, fmt.Errorf("marshal tags: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO skills (name, description, steps_json, tags_json, version, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		skill.Name, skill.Description, string(stepsJSON), string(tagsJSON), skill.Version, skill.CreatedAt.Unix(),
	)
	if err != nil {
		return domain.Skill{}, fmt.Errorf("insert skill: %w", err)
	}
	return skill, nil
}

func (s *SQLiteStore) Get(ctx context.Context, name string) (*domain.Skill, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT name, description, steps_json, tags_json, version, created_at FROM skills WHERE name = ?`, name)
	skill, err := scanSkill(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

func (s *SQLiteStore) List(ctx context.Context, tag string) ([]domain.Skill, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, description, steps_json, tags_json, version, created_at FROM skills ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query skills: %w", err)
	}
	defer rows.Close()

	var skills []domain.Skill
	for rows.Next() {
		skill, err := scanSkillRow(rows)
		if err != nil {
			return nil, err
		}
		if tag == "" || containsTag(skill.Tags, tag) {
			skills = append(skills, skill)
		}
	}
	return skills, rows.Err()
}

func (s *SQLiteStore) Delete(ctx context.Context, name string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM skills WHERE name = ?`, name)
	if err != nil {
		return false, fmt.Errorf("delete skill: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSkill(row *sql.Row) (domain.Skill, error) {
	var s domain.Skill
	var stepsJSON, tagsJSON string
	var createdAt int64
	if err := row.Scan(&s.Name, &s.Description, &stepsJSON, &tagsJSON, &s.Version, &createdAt); err != nil {
		return domain.Skill{}, err
	}
	return unpackSkill(s, stepsJSON, tagsJSON, createdAt)
}

func scanSkillRow(rows *sql.Rows) (domain.Skill, error) {
	var s domain.Skill
	var stepsJSON, tagsJSON string
	var createdAt int64
	if err := rows.Scan(&s.Name, &s.Description, &stepsJSON, &tagsJSON, &s.Version, &createdAt); err != nil {
		return domain.Skill{}, fmt.Errorf("scan skill: %w", err)
	}
	return unpackSkill(s, stepsJSON, tagsJSON, createdAt)
}

func unpackSkill(s domain.Skill, stepsJSON, tagsJSON string, createdAt int64) (domain.Skill, error) {
	if err := json.Unmarshal([]byte(stepsJSON), &s.Steps); err != nil {
		return domain.Skill{}, fmt.Errorf("unmarshal steps: %w", err)
	}
	if err := json.Unmarshal([]byte(tagsJSON), &s.Tags); err != nil {
		return domain.Skill{}, fmt.Errorf("unmarshal tags: %w", err)
	}
	s.CreatedAt = time.Unix(createdAt, 0).UTC()
	return s, nil
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
