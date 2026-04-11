// Package domain defines User and Session entities for Sovereign.
package domain

import "time"

// Role is a named permission level.
type Role string

const (
	RoleAdmin      Role = "admin"
	RoleUser       Role = "user"
	RoleRestricted Role = "restricted"
)

// User represents a Cortex identity.
type User struct {
	UserID       string
	Username     string
	Email        string
	PasswordHash string // bcrypt hash — never returned to callers
	Roles        []string
	CreatedAt    time.Time
	LastActive   time.Time
}

// Session represents an authenticated device session with a bearer token.
type Session struct {
	SessionID  string
	UserID     string
	DeviceID   string
	DeviceName string
	Token      string // opaque random hex, 64 chars
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// IsExpired returns true if the session has passed its expiry time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
