// internal/core/domain/models/auth.go
package models

import "time"

// LoginRequest matches the gRPC Login RPC input (auth.proto).
type LoginRequest struct {
	Email       string
	PhoneNumber string
	Password    string
}

// RegisterPatientRequest matches the gRPC RegisterPatient RPC input (auth.proto).
type RegisterPatientRequest struct {
	PhoneNumber string
	Email       string
	Password    string
	FullName    string
	DateOfBirth time.Time
}

type LoginResult struct {
	Message     string
	UserID      string
	AccessToken string
	Role        string
}

type RegisterResult struct {
	Message string
	UserID  string
}
