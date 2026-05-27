package models

import "time"

type AuditEvent struct {
	EventID        string
	EventType      string
	ActorType      string
	ActorID        string
	PatientID      string
	ServiceName    string
	ResourceType   string
	ResourceID     string
	Timestamp      time.Time
	RequestID      string
	CorrelationID  string
	IPAddress      string
	ToolName       string
	ModelName      string
	ProviderName   string
	SuccessStatus  bool
	FailureReason  string
	MetadataJSON   []byte
}

type Consent struct {
	ConsentID      string
	PatientID      string
	ConsentType    string
	GrantedBy      string
	GrantedAt      time.Time
	RevokedAt      *time.Time
	Scope          string
	ExpirationTime *time.Time
	Source         string
	MetadataJSON   []byte
}

type AuditEventFilter struct {
	EventType     string
	ActorType     string
	ActorID       string
	PatientID     string
	ServiceName   string
	CorrelationID string
	Limit         int32
}

type ConsentFilter struct {
	PatientID       string
	ConsentType     string
	IncludeRevoked  bool
	Limit           int32
}

type HospitalIncidentFilter struct {
	HospitalID string
	Limit      int32
}
