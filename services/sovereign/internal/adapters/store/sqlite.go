// Package store provides a SQLite-backed IdentityStore for Sovereign.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/sovereign/internal/domain"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
    user_id       TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    roles         TEXT NOT NULL DEFAULT 'user',
    created_at    INTEGER NOT NULL,
    last_active   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    session_id  TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    device_id   TEXT NOT NULL DEFAULT '',
    device_name TEXT NOT NULL DEFAULT '',
    token       TEXT NOT NULL UNIQUE,
    expires_at  INTEGER NOT NULL,
    created_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_token     ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user      ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires   ON sessions(expires_at);
`

// SQLiteStore persists Sovereign identity data.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sovereign db at %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("sovereign schema migration: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) CreateUser(ctx context.Context, u *domain.User) error {
	if u.UserID == "" {
		u.UserID = uuid.NewString()
	}
	now := time.Now()
	u.CreatedAt = now
	u.LastActive = now
	roles := strings.Join(u.Roles, ",")
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (user_id, username, email, password_hash, roles, created_at, last_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.UserID, u.Username, u.Email, u.PasswordHash, roles,
		now.UnixMilli(), now.UnixMilli(),
	)
	return err
}

func (s *SQLiteStore) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT user_id, username, email, password_hash, roles, created_at, last_active FROM users WHERE user_id = ?`,
		userID)
	return scanUser(row)
}

func (s *SQLiteStore) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT user_id, username, email, password_hash, roles, created_at, last_active FROM users WHERE username = ?`,
		username)
	return scanUser(row)
}

func (s *SQLiteStore) UpdateLastActive(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET last_active = ? WHERE user_id = ?`,
		time.Now().UnixMilli(), userID,
	)
	return err
}

func (s *SQLiteStore) ListUsers(ctx context.Context) ([]*domain.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT user_id, username, email, password_hash, roles, created_at, last_active FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.User
	for rows.Next() {
		u := &domain.User{}
		var rolesStr string
		var createdMs, lastMs int64
		if err := rows.Scan(&u.UserID, &u.Username, &u.Email, &u.PasswordHash, &rolesStr, &createdMs, &lastMs); err != nil {
			continue
		}
		u.Roles = strings.Split(rolesStr, ",")
		u.CreatedAt = time.UnixMilli(createdMs)
		u.LastActive = time.UnixMilli(lastMs)
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CreateSession(ctx context.Context, sess *domain.Session) error {
	if sess.SessionID == "" {
		sess.SessionID = uuid.NewString()
	}
	if sess.Token == "" {
		sess.Token = GenerateToken()
	}
	sess.CreatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (session_id, user_id, device_id, device_name, token, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.SessionID, sess.UserID, sess.DeviceID, sess.DeviceName,
		sess.Token, sess.ExpiresAt.UnixMilli(), sess.CreatedAt.UnixMilli(),
	)
	return err
}

func (s *SQLiteStore) GetSessionByToken(ctx context.Context, token string) (*domain.Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT session_id, user_id, device_id, device_name, token, expires_at, created_at
		 FROM sessions WHERE token = ?`, token)
	sess := &domain.Session{}
	var expiresMs, createdMs int64
	err := row.Scan(&sess.SessionID, &sess.UserID, &sess.DeviceID, &sess.DeviceName,
		&sess.Token, &expiresMs, &createdMs)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess.ExpiresAt = time.UnixMilli(expiresMs)
	sess.CreatedAt = time.UnixMilli(createdMs)
	return sess, nil
}

func (s *SQLiteStore) RevokeSession(ctx context.Context, token string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLiteStore) PruneExpired(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < ?`, time.Now().UnixMilli())
	return err
}

func scanUser(row *sql.Row) (*domain.User, error) {
	u := &domain.User{}
	var rolesStr string
	var createdMs, lastMs int64
	err := row.Scan(&u.UserID, &u.Username, &u.Email, &u.PasswordHash, &rolesStr, &createdMs, &lastMs)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Roles = strings.Split(rolesStr, ",")
	u.CreatedAt = time.UnixMilli(createdMs)
	u.LastActive = time.UnixMilli(lastMs)
	return u, nil
}

// GenerateToken creates a cryptographically random 32-byte opaque token.
func GenerateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("sovereign: failed to generate token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

