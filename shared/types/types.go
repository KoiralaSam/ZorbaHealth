package types

import "time"

// PaginationParams contains pagination parameters
type PaginationParams struct {
	Page     int `json:"page" query:"page"`
	PageSize int `json:"page_size" query:"page_size"`
	Offset   int `json:"-"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
}

type CallEvent struct {
	EventType CallStatus // "call.started" | "call.ended"
	PatientID string
	SessionID string
}

// CallStatus represents the status of a voice call
type CallStatus string

const (
	CallStatusPending    CallStatus = "pending"
	CallStatusRinging    CallStatus = "ringing"
	CallStatusConnecting CallStatus = "connecting"
	CallStatusActive     CallStatus = "active"
	CallStatusEnded      CallStatus = "ended"
	CallStatusFailed     CallStatus = "failed"
	CallStatusCancelled  CallStatus = "cancelled"
)

// Priority represents task or item priority
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// HealthMetric represents a single health measurement
type HealthMetric struct {
	Type      string    `json:"type"`  // e.g., "blood_pressure", "heart_rate", "temperature"
	Value     string    `json:"value"` // e.g., "120/80", "72", "98.6"
	Unit      string    `json:"unit"`  // e.g., "mmHg", "bpm", "°F"
	Timestamp time.Time `json:"timestamp"`
	Notes     string    `json:"notes,omitempty"`
}

// TimeRange represents a time period
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Metadata represents flexible key-value metadata
type Metadata map[string]interface{}

// SMSRequest is the payload for the VoIP.ms SMS/MMS URL callback (GET query params or POST body).
// When SMS/MMS is received by your DID, VoIP.ms sends these fields to your webhook.
type SMSRequest struct {
	ID      string `json:"id"`      // The ID of the SMS/MMS message
	Date    string `json:"date"`    // The date and time the message was received (timestamp)
	From    string `json:"from"`    // The phone number that sent the message
	To      string `json:"to"`      // The DID number that received the message
	Message string `json:"message"` // The content of the message
	Files   string `json:"files"`   // Comma-separated list of media files (MMS)
}

// SMSResponse is the body you may return from your SMS webhook (e.g. 200 OK).
// VoIP.ms docs do not specify a required response shape; this is for consistency if you send JSON back.
type SMSResponse struct {
	Status  string `json:"status,omitempty"`  // e.g. "ok", "received"
	Message string `json:"message,omitempty"` // Optional human-readable message
}

const LocationUpdateType = "location_update"

// LocationUpdate is the JSON payload sent over the location WebSocket.
// Example:
//
//	{
//	  "type": "location_update",
//	  "sessionID": "abc123",
//	  "lat": 27.7,
//	  "lng": 85.3,
//	  "accuracy": 12.5
//	}
type LocationUpdate struct {
	Type      string  `json:"type,omitempty"`
	SessionID string  `json:"sessionID"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Accuracy  float64 `json:"accuracy"`
}
