// Package bridge defines the shared Redis contract for bridged patient-staff
// call sessions. patient-service writes sessions under Key(sessionID) and
// interpretation-service reads them, so both must agree on this JSON shape.
//
// Field names intentionally match the historical encoding produced by
// marshaling the patient-service domain model directly (Go default exported
// field names), keeping existing Redis payloads readable.
package bridge

import "time"

// KeyPrefix is the Redis key namespace for bridged call sessions.
const KeyPrefix = "voice:bridge:"

// Key returns the Redis key for a bridged call session.
func Key(sessionID string) string {
	return KeyPrefix + sessionID
}

// Session status values.
const (
	StatusTransferRequested = "transfer_requested"
	StatusConnected         = "connected"
	StatusEnded             = "ended"
)

// Participant identifiers used by the interpretation relay.
const (
	ParticipantPatient = "patient"
	ParticipantStaff   = "staff"
)

// TranslationPreferences holds per-party interpretation settings.
type TranslationPreferences struct {
	Enabled             bool
	LanguageMode        string // auto | manual
	LanguageCode        string
	ParticipantIdentity string
	UpdatedAt           time.Time
}

// Session is the Redis JSON payload for one bridged call.
type Session struct {
	SessionID            string
	RoomSID              string
	PatientID            string
	HospitalID           string
	StaffID              string
	PatientAccessToken   string
	StaffAccessToken     string
	Status               string
	RequestedByActorType string
	RequestedByActorID   string
	TransferReason       string
	RequestedAt          time.Time
	ConnectedAt          *time.Time
	EndedAt              *time.Time
	PatientTranslation   TranslationPreferences
	StaffTranslation     TranslationPreferences
}
