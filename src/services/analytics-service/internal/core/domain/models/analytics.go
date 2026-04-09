package models

import "time"

// HospitalSummary is the top-level dashboard card for a hospital admin.
type HospitalSummary struct {
	HospitalID             string
	TotalConsentedPatients int
	TotalCalls30d          int
	EmergencyEvents30d     int
	AvgCallDurationSeconds float64
	RecordsIndexed         int
	ActivePatients7d       int
}

// CallVolumePoint is a single data point for a time-series chart.
type CallVolumePoint struct {
	Date           time.Time
	TotalCalls     int
	CompletedCalls int
	EmergencyCalls int
	AvgDurationSec float64
}

// ConditionCount represents how prevalent a condition is across
// a hospital's consented patient population.
type ConditionCount struct {
	ConditionName string
	PatientCount  int
	Percentage    float64 // of total consented patients
}

// ToolUsageStat shows how often each MCP tool was called and its outcomes.
type ToolUsageStat struct {
	Tool         string
	SuccessCount int
	ErrorCount   int
	DeniedCount  int
	SuccessRate  float64
}

// ActivityEvent is a single entry in the recent activity feed.
type ActivityEvent struct {
	Timestamp time.Time
	Tool      string
	ActorType string // "patient" | "staff"
	Outcome   string
	SessionID string
}

// PlatformSummary is the system-wide view for Zorba admins.
type PlatformSummary struct {
	TotalHospitals      int
	TotalPatients       int
	TotalCalls30d       int
	TotalEmergencies30d int
	AvgCallDurationSec  float64
	ActiveHospitals7d   int
}

// HospitalStat is a row in the per-hospital breakdown table.
type HospitalStat struct {
	HospitalID     string
	HospitalName   string
	PatientCount   int
	CallCount      int
	EmergencyCount int
}

// PatientCall is a single call record from a patient's own history.
type PatientCall struct {
	StartedAt    time.Time
	Duration     string
	Summary      string
	HadEmergency bool
}

// PatientCallHistory is the patient's complete call history summary.
type PatientCallHistory struct {
	TotalCalls int
	Calls      []PatientCall
}
