package models

// Auth represents an authentication session for a user.
// The concrete persistence (table structure, indexes, etc.) lives in the auth-service's database layer.
type Auth struct {
	ID       uint64
	UserID   string
	AuthUUID string
}
