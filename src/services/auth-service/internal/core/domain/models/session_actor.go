package models

type SessionActor struct {
	ActorType  string
	PatientID  string
	SessionID  string
	StaffID    string
	HospitalID string
	Role       string
	AdminID    string
	Scopes     []string
}
