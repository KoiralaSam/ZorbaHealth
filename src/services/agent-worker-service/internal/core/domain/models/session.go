package models

import "time"

type SessionStart struct {
	RoomName      string
	SessionID     string
	Language      string
	CallerPhone   string
	PatientIDHint string
}

type IdentifiedPatient struct {
	PatientID string
}

type PatientCandidate struct {
	PatientID string
	FullName  string
}

type RegistrationRequest struct {
	PhoneNumber string
	Email       string
	FullName    string
	DateOfBirth time.Time
}

type SessionState struct {
	RoomName             string
	SessionID            string
	Language             string
	CallerPhone          string
	PatientCandidate     *PatientCandidate
	Patient              *IdentifiedPatient
	Token                string
	Context              string
	PhoneLookupAttempted bool
	VerificationMode     string
	RegistrationToken    string
	IdentityGateEnabled  bool
	IdentityGateSticky   bool

	// Echo suppression / de-dupe controls.
	SpeakingUntil     time.Time
	LastAssistantText string
	LastUserText      string
	LastUserAt        time.Time
}

type Message struct {
	Role       string
	Content    string
	ToolCall   *ToolCall
	ToolCallID string
}

type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

type ChatResponse struct {
	Text    string
	ToolUse *ToolCall
}

type ToolDef struct {
	Name        string
	Description string
	InputSchema map[string]any
}
