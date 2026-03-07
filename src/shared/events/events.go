package events

const (
	NotifyPatientRegisteredQueue        = "notify_patient_registered"
	NotifyPatientPendingVerificationQueue = "notify_patient_pending_verification"
)

// PatientRegisteredData is the payload when a patient has completed registration (e.g. after email verification).
type PatientRegisteredData struct {
	Message   string `json:"message"`
	PatientID string `json:"patient_id"`
	UserID    string `json:"user_id"`
}

// PendingRegistrationData is the payload for pending (pre-verification) registration events. No password.
type PendingRegistrationData struct {
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	FullName    string `json:"full_name"`
	DateOfBirth string `json:"date_of_birth"` // ISO8601
}

// PatientEventData is the envelope for patient-related events.
type PatientEventData struct {
	Patient         *PatientRegisteredData     `json:"patient,omitempty"`
	RegisterRequest *PendingRegistrationData   `json:"register_request,omitempty"`
}
