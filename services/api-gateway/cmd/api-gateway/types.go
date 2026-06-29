package main

import "time"

type PatientLoginRequest struct {
	Identifier  string `json:"identifier"`
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type PatientLoginResponse struct {
	Message      string `json:"message,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	PatientID    string `json:"patient_id,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
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

type BridgedCallTranslationPreferencesRecord struct {
	Enabled             bool   `json:"enabled"`
	LanguageMode        string `json:"language_mode,omitempty"`
	LanguageCode        string `json:"language_code,omitempty"`
	ParticipantIdentity string `json:"participant_identity,omitempty"`
	UpdatedAt           string `json:"updated_at,omitempty"`
}

type BridgedCallSessionRecord struct {
	SessionID            string                                  `json:"session_id,omitempty"`
	RoomSID              string                                  `json:"room_sid,omitempty"`
	PatientID            string                                  `json:"patient_id,omitempty"`
	HospitalID           string                                  `json:"hospital_id,omitempty"`
	StaffID              string                                  `json:"staff_id,omitempty"`
	Status               string                                  `json:"status,omitempty"`
	RequestedByActorType string                                  `json:"requested_by_actor_type,omitempty"`
	RequestedByActorID   string                                  `json:"requested_by_actor_id,omitempty"`
	TransferReason       string                                  `json:"transfer_reason,omitempty"`
	RequestedAt          string                                  `json:"requested_at,omitempty"`
	ConnectedAt          string                                  `json:"connected_at,omitempty"`
	EndedAt              string                                  `json:"ended_at,omitempty"`
	PatientTranslation   BridgedCallTranslationPreferencesRecord `json:"patient_translation"`
	StaffTranslation     BridgedCallTranslationPreferencesRecord `json:"staff_translation"`
}

type BridgedCallSessionResponse struct {
	Session          BridgedCallSessionRecord `json:"session"`
	PatientRoomToken string                   `json:"patient_room_token,omitempty"`
	LiveKitWSURL     string                   `json:"livekit_ws_url,omitempty"`
}

type BridgedCallConnectResponse struct {
	Session BridgedCallSessionRecord `json:"session"`
	// LiveKit join credentials for the connecting staff member.
	StaffRoomToken string `json:"staff_room_token,omitempty"`
	LiveKitWSURL   string `json:"livekit_ws_url,omitempty"`
}

type BridgedCallSessionListResponse struct {
	Sessions []BridgedCallSessionRecord `json:"sessions"`
}

type RequestBridgedCallTransferRequest struct {
	SessionID      string `json:"session_id"`
	RoomSID        string `json:"room_sid,omitempty"`
	HospitalID     string `json:"hospital_id"`
	StaffID        string `json:"staff_id,omitempty"`
	TransferReason string `json:"transfer_reason,omitempty"`
}

type ConnectBridgedCallRequest struct {
	SessionID                string `json:"session_id"`
	StaffParticipantIdentity string `json:"staff_participant_identity,omitempty"`
}

type UpdateBridgedCallTranslationRequest struct {
	SessionID   string                                  `json:"session_id"`
	Participant string                                  `json:"participant"`
	Translation BridgedCallTranslationPreferencesRecord `json:"translation"`
}

type EndBridgedCallRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
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

type HospitalRegisterRequest struct {
	HospitalName       string `json:"hospital_name"`
	LicenseNo          string `json:"license_no"`
	Email              string `json:"email"`
	PhoneNumber        string `json:"phone_number,omitempty"`
	Password           string `json:"password"`
	StaffName          string `json:"staff_name"`
	StaffRole          string `json:"staff_role,omitempty"`
	Address            string `json:"address,omitempty"`
	RegistrationNumber string `json:"registration_number,omitempty"`
}

type HospitalStaffRegisterRequest struct {
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Password    string `json:"password"`
	StaffName   string `json:"staff_name"`
	StaffRole   string `json:"staff_role"`
}

type HospitalRegisterResponse struct {
	Message    string `json:"message,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	HospitalID string `json:"hospital_id,omitempty"`
	StaffID    string `json:"staff_id,omitempty"`
	StaffRole  string `json:"staff_role,omitempty"`
}

type HospitalLoginResponse struct {
	Message      string `json:"message,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	HospitalID   string `json:"hospital_id,omitempty"`
	StaffID      string `json:"staff_id,omitempty"`
	Role         string `json:"role,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type HospitalPatientSummaryRequest struct {
	PatientID string `json:"patient_id"`
	Focus     string `json:"focus,omitempty"`
}

type HospitalPatientSummaryResponse struct {
	PatientID string `json:"patient_id,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

type HospitalPatientRecord struct {
	PatientID        string `json:"patient_id,omitempty"`
	FullName         string `json:"full_name,omitempty"`
	Email            string `json:"email,omitempty"`
	PhoneNumber      string `json:"phone_number,omitempty"`
	DateOfBirth      string `json:"date_of_birth,omitempty"`
	ConsentGrantedAt string `json:"consent_granted_at,omitempty"`
	LastCallAt       string `json:"last_call_at,omitempty"`
}

type HospitalPatientListResponse struct {
	Patients []HospitalPatientRecord `json:"patients"`
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

type HospitalConsentRequestCreateRequest struct {
	RequestedPermissions []string `json:"requested_permissions,omitempty"`
	Note                 string   `json:"note,omitempty"`
	ExpiresInMinutes     int      `json:"expires_in_minutes,omitempty"`
}

type HospitalConsentRequestRecord struct {
	ID                   string   `json:"id,omitempty"`
	Token                string   `json:"token,omitempty"`
	HospitalID           string   `json:"hospital_id,omitempty"`
	HospitalName         string   `json:"hospital_name,omitempty"`
	StaffID              string   `json:"staff_id,omitempty"`
	StaffName            string   `json:"staff_name,omitempty"`
	StaffRole            string   `json:"staff_role,omitempty"`
	PatientID            string   `json:"patient_id,omitempty"`
	RequestedPermissions []string `json:"requested_permissions,omitempty"`
	Note                 string   `json:"note,omitempty"`
	ExpiresAt            string   `json:"expires_at,omitempty"`
	ApprovedAt           string   `json:"approved_at,omitempty"`
	CreatedAt            string   `json:"created_at,omitempty"`
	Status               string   `json:"status,omitempty"`
	QRPayload            string   `json:"qr_payload,omitempty"`
}

type HospitalConsentRequestCreateResponse struct {
	Request HospitalConsentRequestRecord `json:"request"`
}

type HospitalConsentRequestListResponse struct {
	Requests []HospitalConsentRequestRecord `json:"requests"`
}

type PatientHospitalConsentRecord struct {
	HospitalID   string `json:"hospital_id,omitempty"`
	HospitalName string `json:"hospital_name,omitempty"`
	GrantedAt    string `json:"granted_at,omitempty"`
	RevokedAt    string `json:"revoked_at,omitempty"`
	Status       string `json:"status,omitempty"`
}

type PatientHospitalConsentListResponse struct {
	Consents []PatientHospitalConsentRecord `json:"consents"`
}

type PatientConsentRequestLookupResponse struct {
	Request HospitalConsentRequestRecord `json:"request"`
}

type PatientConsentRequestApproveResponse struct {
	Message string                       `json:"message,omitempty"`
	Consent PatientHospitalConsentRecord `json:"consent"`
}
