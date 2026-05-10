package models

import "time"

// User represents a user in the system.
// The concrete persistence (table structure, indexes, etc.) lives in the auth-service's database layer.
type User struct {
	ID           string
	Email        string
	PhoneNumber  string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
