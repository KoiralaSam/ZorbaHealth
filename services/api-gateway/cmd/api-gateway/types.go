package main

import "time"

type PatientLoginRequest struct {
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type PatientLoginResponse struct {
	Message     string `json:"message,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	PatientID   string `json:"patient_id,omitempty"`
}

type PatientProfileResponse struct {
	PatientID     string `json:"patient_id,omitempty"`
	FullName      string `json:"full_name,omitempty"`
	Email         string `json:"email,omitempty"`
	PhoneNumber   string `json:"phone_number,omitempty"`
	DateOfBirth   string `json:"date_of_birth,omitempty"`
	MedicalNotes  string `json:"medical_notes,omitempty"`
	VoicePhone    string `json:"voice_phone,omitempty"`
	VoiceEnabled  bool   `json:"voice_enabled,omitempty"`
	SupportWindow string `json:"support_window,omitempty"`
}

type ConsentRecord struct {
	ConsentID      string         `json:"consent_id,omitempty"`
	ConsentType    string         `json:"consent_type,omitempty"`
	GrantedBy      string         `json:"granted_by,omitempty"`
	GrantedAt      string         `json:"granted_at,omitempty"`
	RevokedAt      string         `json:"revoked_at,omitempty"`
	Scope          string         `json:"scope,omitempty"`
	ExpirationTime string         `json:"expiration_time,omitempty"`
	Source         string         `json:"source,omitempty"`
	Status         string         `json:"status,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type PatientConsentListResponse struct {
	PatientID string          `json:"patient_id,omitempty"`
	Consents  []ConsentRecord `json:"consents,omitempty"`
}

type PatientConsentMutationRequest struct {
	ConsentType string         `json:"consent_type"`
	Scope       string         `json:"scope,omitempty"`
	Source      string         `json:"source,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type PatientConsentMutationResponse struct {
	Message string        `json:"message,omitempty"`
	Consent ConsentRecord `json:"consent,omitempty"`
}

type PatientHealthAnswerRequest struct {
	Question string `json:"question"`
	TopK     int32  `json:"top_k,omitempty"`
}

type PatientHealthCitation struct {
	Text       string  `json:"text,omitempty"`
	SourceFile string  `json:"source_file,omitempty"`
	Score      float32 `json:"score,omitempty"`
}

type PatientHealthAnswerResponse struct {
	Answer    string                  `json:"answer,omitempty"`
	Citations []PatientHealthCitation `json:"citations,omitempty"`
}

type PatientCallSummary struct {
	ID            int64  `json:"id"`
	Status        string `json:"status,omitempty"`
	StartedAt     string `json:"started_at,omitempty"`
	EndedAt       string `json:"ended_at,omitempty"`
	Summary       string `json:"summary,omitempty"`
	RecordingURL  string `json:"recording_url,omitempty"`
	LivekitRoomID string `json:"livekit_room_id,omitempty"`
}

type PatientCallListResponse struct {
	PatientID string               `json:"patient_id,omitempty"`
	Calls     []PatientCallSummary `json:"calls,omitempty"`
}

type CreateWelfareCheckRequest struct {
	ScheduledAt  string `json:"scheduled_at"`
	Timezone     string `json:"timezone"`
	ReasonCode   string `json:"reason_code"`
	ReasonDetail string `json:"reason_detail,omitempty"`
}

type WelfareCheckRecord struct {
	ID                     string `json:"id"`
	PatientID              string `json:"patient_id"`
	ScheduledAt            string `json:"scheduled_at"`
	Timezone               string `json:"timezone"`
	ReasonCode             string `json:"reason_code"`
	ReasonDetail           string `json:"reason_detail,omitempty"`
	Status                 string `json:"status"`
	RecurrenceRule         string `json:"recurrence_rule,omitempty"`
	CreatedAt              string `json:"created_at,omitempty"`
	UpdatedAt              string `json:"updated_at,omitempty"`
	CancelledAt            string `json:"cancelled_at,omitempty"`
	LatestRunID            string `json:"latest_run_id,omitempty"`
	LatestRunStatus        string `json:"latest_run_status,omitempty"`
	LatestRunAttempts      int32  `json:"latest_run_attempts,omitempty"`
	LatestRunFailureReason string `json:"latest_run_failure_reason,omitempty"`
}

type WelfareCheckResponse struct {
	WelfareCheck WelfareCheckRecord `json:"welfare_check"`
}

type WelfareCheckListResponse struct {
	WelfareChecks []WelfareCheckRecord `json:"welfare_checks"`
}

type AuditEventRecord struct {
	EventID       string         `json:"event_id,omitempty"`
	EventType     string         `json:"event_type,omitempty"`
	ActorType     string         `json:"actor_type,omitempty"`
	ActorID       string         `json:"actor_id,omitempty"`
	PatientID     string         `json:"patient_id,omitempty"`
	ServiceName   string         `json:"service_name,omitempty"`
	ResourceType  string         `json:"resource_type,omitempty"`
	ResourceID    string         `json:"resource_id,omitempty"`
	Timestamp     string         `json:"timestamp,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	ToolName      string         `json:"tool_name,omitempty"`
	SuccessStatus bool           `json:"success_status"`
	FailureReason string         `json:"failure_reason,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type PatientAuditResponse struct {
	PatientID string             `json:"patient_id,omitempty"`
	Events    []AuditEventRecord `json:"events,omitempty"`
}

type PatientRegisterRequest struct {
	PhoneNumber string    `json:"phone_number"`
	Email       string    `json:"email"`
	Password    string    `json:"password"`
	FullName    string    `json:"full_name"`
	DateOfBirth time.Time `json:"date_of_birth"`
}

type PatientRegisterResponse struct {
	Message   string `json:"message,omitempty"`
	PatientID string `json:"patient_id,omitempty"`
}

type PatientRegisterVerifyOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	OTP         string `json:"otp"`
}

type HospitalLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type HospitalLoginResponse struct {
	Message     string `json:"message,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	HospitalID  string `json:"hospital_id,omitempty"`
	Role        string `json:"role,omitempty"`
}

type HospitalPatientSummaryRequest struct {
	PatientID string `json:"patient_id"`
	Focus     string `json:"focus,omitempty"`
}

type HospitalPatientSummaryResponse struct {
	Summary string `json:"summary,omitempty"`
}

type HospitalIncidentRecord struct {
	EventID       string         `json:"event_id,omitempty"`
	PatientID     string         `json:"patient_id,omitempty"`
	Timestamp     string         `json:"timestamp,omitempty"`
	Severity      string         `json:"severity,omitempty"`
	SessionID     string         `json:"session_id,omitempty"`
	ServiceName   string         `json:"service_name,omitempty"`
	FailureReason string         `json:"failure_reason,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type HospitalIncidentListResponse struct {
	Incidents []HospitalIncidentRecord `json:"incidents,omitempty"`
}

type HospitalPatientAuditResponse struct {
	PatientID string             `json:"patient_id,omitempty"`
	Events    []AuditEventRecord `json:"events,omitempty"`
}
