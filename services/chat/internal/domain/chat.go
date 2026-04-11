package domain

// Chat is a named, persistent conversation owned by a user.
type Chat struct {
	ChatID        string
	UserID        string
	WorkspaceID   string
	Title         string
	CreatedAt     int64 // unix millis
	LastMessageAt int64 // unix millis, 0 if no messages
}

// Message is a single turn in a chat — either user or assistant.
type Message struct {
	MessageID string
	ChatID    string
	Role      string // "user" | "assistant"
	Content   string
	CreatedAt int64 // unix millis
}
