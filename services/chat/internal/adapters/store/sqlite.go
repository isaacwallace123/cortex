// Package store provides a SQLite-backed Chat storage adapter.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/chat/internal/domain"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS chats (
    chat_id         TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    workspace_id    TEXT NOT NULL DEFAULT '',
    title           TEXT NOT NULL,
    created_at      INTEGER NOT NULL,
    last_message_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS chats_user_idx ON chats(user_id, workspace_id, last_message_at DESC);

CREATE TABLE IF NOT EXISTS messages (
    message_id TEXT PRIMARY KEY,
    chat_id    TEXT NOT NULL REFERENCES chats(chat_id) ON DELETE CASCADE,
    role       TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS messages_chat_idx ON messages(chat_id, created_at ASC);
`

// SQLiteStore persists chats and messages in SQLite.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open chat db at %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("chat schema migration: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) CreateChat(ctx context.Context, chat domain.Chat) error {
	if chat.ChatID == "" {
		chat.ChatID = uuid.New().String()
	}
	if chat.CreatedAt == 0 {
		chat.CreatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO chats(chat_id, user_id, workspace_id, title, created_at, last_message_at)
		 VALUES (?,?,?,?,?,?)`,
		chat.ChatID, chat.UserID, chat.WorkspaceID, chat.Title, chat.CreatedAt, chat.LastMessageAt,
	)
	return err
}

func (s *SQLiteStore) GetChat(ctx context.Context, chatID, userID string) (domain.Chat, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT chat_id, user_id, workspace_id, title, created_at, last_message_at
		 FROM chats WHERE chat_id=? AND user_id=?`, chatID, userID)
	var c domain.Chat
	if err := row.Scan(&c.ChatID, &c.UserID, &c.WorkspaceID, &c.Title, &c.CreatedAt, &c.LastMessageAt); err != nil {
		if err == sql.ErrNoRows {
			return domain.Chat{}, fmt.Errorf("chat not found: %s", chatID)
		}
		return domain.Chat{}, err
	}
	return c, nil
}

func (s *SQLiteStore) ListChats(ctx context.Context, userID, workspaceID string, limit int) ([]domain.Chat, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if workspaceID != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT chat_id, user_id, workspace_id, title, created_at, last_message_at
			 FROM chats WHERE user_id=? AND workspace_id=?
			 ORDER BY last_message_at DESC, created_at DESC LIMIT ?`,
			userID, workspaceID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT chat_id, user_id, workspace_id, title, created_at, last_message_at
			 FROM chats WHERE user_id=?
			 ORDER BY last_message_at DESC, created_at DESC LIMIT ?`,
			userID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chats []domain.Chat
	for rows.Next() {
		var c domain.Chat
		if err := rows.Scan(&c.ChatID, &c.UserID, &c.WorkspaceID, &c.Title, &c.CreatedAt, &c.LastMessageAt); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

func (s *SQLiteStore) DeleteChat(ctx context.Context, chatID, userID string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM chats WHERE chat_id=? AND user_id=?`, chatID, userID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLiteStore) AppendMessage(ctx context.Context, msg domain.Message) error {
	if msg.MessageID == "" {
		msg.MessageID = uuid.New().String()
	}
	if msg.CreatedAt == 0 {
		msg.CreatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages(message_id, chat_id, role, content, created_at) VALUES (?,?,?,?,?)`,
		msg.MessageID, msg.ChatID, msg.Role, msg.Content, msg.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetMessages(ctx context.Context, chatID string, limit int) ([]domain.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT message_id, chat_id, role, content, created_at
		 FROM messages WHERE chat_id=? ORDER BY created_at ASC LIMIT ?`, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []domain.Message
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.MessageID, &m.ChatID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *SQLiteStore) UpdateLastMessageAt(ctx context.Context, chatID string, ts int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE chats SET last_message_at=? WHERE chat_id=?`, ts, chatID)
	return err
}
