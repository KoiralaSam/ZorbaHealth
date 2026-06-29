package models

import "time"

type BridgedCallStatus string

const (
	BridgedCallStatusTransferRequested BridgedCallStatus = "transfer_requested"
	BridgedCallStatusConnected         BridgedCallStatus = "connected"
	BridgedCallStatusEnded             BridgedCallStatus = "ended"
)

type BridgedCallParticipant string

const (
	BridgedCallParticipantPatient BridgedCallParticipant = "patient"
	BridgedCallParticipantStaff   BridgedCallParticipant = "staff"
)

type TranslationMode string

const (
	TranslationModeAuto   TranslationMode = "auto"
	TranslationModeManual TranslationMode = "manual"
)

type BridgedCallTranslationPreferences struct {
	Enabled             bool
	LanguageMode        TranslationMode
	LanguageCode        string
	ParticipantIdentity string
	UpdatedAt           time.Time
}

type BridgedCallSession struct {
	SessionID            string
	RoomSID              string
	PatientID            string
	HospitalID           string
	StaffID              string
	PatientAccessToken   string
	StaffAccessToken     string
	Status               BridgedCallStatus
	RequestedByActorType string
	RequestedByActorID   string
	TransferReason       string
	RequestedAt          time.Time
	ConnectedAt          *time.Time
	EndedAt              *time.Time
	PatientTranslation   BridgedCallTranslationPreferences
	StaffTranslation     BridgedCallTranslationPreferences
}

// BridgedCallConnectResult carries the updated session plus a freshly minted
// LiveKit join token for the connecting staff member (never persisted).
type BridgedCallConnectResult struct {
	Session        *BridgedCallSession
	StaffRoomToken string
	LiveKitWSURL   string
}

type RequestBridgedCallTransferCommand struct {
	SessionID   string
	RoomSID     string
	PatientID   string
	HospitalID  string
	StaffID     string
	Reason      string
	ActorType   string
	ActorID     string
	AccessToken string
}

type UpdateBridgedCallTranslationCommand struct {
	SessionID   string
	Participant BridgedCallParticipant
	Preferences BridgedCallTranslationPreferences
	ActorType   string
	ActorID     string
	StaffID     string
	HospitalID  string
	PatientID   string
	AccessToken string
}
