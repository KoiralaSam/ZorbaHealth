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
