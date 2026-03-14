package models

import (
	"time"

	"github.com/google/uuid"
)

type Patient struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	PhoneNumber  string
	Email        string
	FullName     string    `json:"full_name"`
	DateOfBirth  time.Time `json:"date_of_birth"`
	MedicalNotes string    `json:"medical_notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PendingRegistration is stored in Redis until the user verifies. Do not persist password in plain text in production; hash before storing or use short TTL.
type PendingRegistration struct {
	Email          string    `json:"email"`
	PhoneNumber    string    `json:"phone_number"`
	Password       string    `json:"password"` // prefer hashed; or omit and require reset on first login
	FullName       string    `json:"full_name"`
	DateOfBirth    time.Time `json:"date_of_birth"`
	CreatedAt      time.Time `json:"created_at"`
	PhoneVerified  bool      `json:"phone_verified"`
	EmailVerified  bool      `json:"email_verified"`
}
