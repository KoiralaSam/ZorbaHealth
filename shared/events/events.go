package events

import "github.com/KoiralaSam/ZorbaHealth/shared/contracts"

const (
	// PatientExchange is the topic exchange for patient-related events (publish + consume).
	PatientExchange    = "patient"
	EscalationExchange = "zorba.escalation"

	NotifyPatientRegisteredQueue          = "notify_patient_registered"
	NotifyPatientPendingVerificationQueue = "notify_patient_pending_verification"
	NotifyEmergencyEscalationQueue        = "notify_emergency_escalation"
	NotifyMeetingRequestedQueue           = "notify_meeting_requested"
	NotifyMeetingScheduledQueue           = "notify_meeting_scheduled"
	NotifyMeetingReminderQueue            = "notify_meeting_reminder"
	NotifyAppointmentBookedQueue          = "notify_appointment_booked"
	NotifyAppointmentCancelledQueue       = "notify_appointment_cancelled"

	// CallsExchange is the topic exchange for call lifecycle events.
	CallsExchange = "zorba.calls"

	// LocationCallEventsQueue is consumed by location-service for call.* events.
	LocationCallEventsQueue = "location.call_events"
)

// QueueBinding describes a queue and its routing keys when binding to a topic exchange.
type QueueBinding struct {
	QueueName   string
	RoutingKeys []string
}

// PatientPublisherQueueBindings — patient-service publishes only; declares PatientExchange with no consumer queues.
var PatientPublisherQueueBindings = []QueueBinding{}

// AuthServicePatientQueueBindings — auth-service consumes registered/updated patient events.
var AuthServicePatientQueueBindings = []QueueBinding{
	{
		QueueName:   NotifyPatientRegisteredQueue,
		RoutingKeys: []string{contracts.PatientEventRegistered, contracts.PatientEventUpdated},
	},
}

// NotificationServicePatientQueueBindings — notification-service consumes pending verification (chached) events.
var NotificationServicePatientQueueBindings = []QueueBinding{
	{
		QueueName:   NotifyPatientPendingVerificationQueue,
		RoutingKeys: []string{contracts.PatientEventChached, contracts.PatientEventVerificationCodeRequested},
	},
	{
		QueueName:   NotifyEmergencyEscalationQueue,
		RoutingKeys: []string{contracts.EmergencyEscalatedEvent},
	},
	{
		QueueName:   NotifyMeetingRequestedQueue,
		RoutingKeys: []string{contracts.PatientEventMeetingRequested},
	},
	{
		QueueName:   NotifyMeetingScheduledQueue,
		RoutingKeys: []string{contracts.PatientEventMeetingScheduled},
	},
	{
		QueueName:   NotifyMeetingReminderQueue,
		RoutingKeys: []string{contracts.PatientEventMeetingReminder},
	},
	{
		QueueName:   NotifyAppointmentBookedQueue,
		RoutingKeys: []string{contracts.AppointmentEventBooked},
	},
	{
		QueueName:   NotifyAppointmentCancelledQueue,
		RoutingKeys: []string{contracts.AppointmentEventCancelled},
	},
}

// LocationServiceCallsQueueBindings — location-service consumes call lifecycle events.
// This uses topic semantics: "call.*" matches "call.started", "call.ended", etc.
var LocationServiceCallsQueueBindings = []QueueBinding{
	{
		QueueName:   LocationCallEventsQueue,
		RoutingKeys: []string{contracts.CallEventAll},
	},
}

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
	Otp         string `json:"otp,omitempty"` // OTP sent via SMS for phone verification
}

type PhoneVerificationData struct {
	PhoneNumber string `json:"phone_number"`
	FullName    string `json:"full_name"`
	Otp         string `json:"otp,omitempty"`
}

// PatientEventData is the envelope for patient-related events.
type PatientEventData struct {
	Patient           *PatientRegisteredData   `json:"patient,omitempty"`
	RegisterRequest   *PendingRegistrationData `json:"register_request,omitempty"`
	PhoneVerification *PhoneVerificationData   `json:"phone_verification,omitempty"`
}

// CallEvent represents a call lifecycle event that should trigger location handling.
type CallEvent struct {
	EventType string `json:"event_type"` // e.g. "call.started" | "call.ended"
	PatientID string `json:"patient_id"`
	SessionID string `json:"session_id"`
}

type EmergencyEscalationData struct {
	SessionID         string   `json:"session_id"`
	PatientID         string   `json:"patient_id,omitempty"`
	CallerPhone       string   `json:"caller_phone,omitempty"`
	Reason            string   `json:"reason"`
	Severity          string   `json:"severity,omitempty"`
	TransferRequested bool     `json:"transfer_requested,omitempty"`
	TransferTarget    string   `json:"transfer_target,omitempty"`
	AlertPhoneNumbers []string `json:"alert_phone_numbers,omitempty"`
	TranscriptExcerpt string   `json:"transcript_excerpt,omitempty"`
}

// MeetingRequestedData is published after a meeting request is persisted but before LiveKit join links exist.
type MeetingRequestedData struct {
	MeetingID       string `json:"meeting_id"`
	PatientID       string `json:"patient_id"`
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	CorrelationID   string `json:"correlation_id"`
	StartsAtRFC3339 string `json:"starts_at"`
	DurationMinutes int    `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title"`
	PatientName     string `json:"patient_name"`
	StaffEmail      string `json:"staff_email"`
	StaffName       string `json:"staff_name"`
}

// MeetingScheduledData is published after a health-staff meeting is persisted.
type MeetingScheduledData struct {
	MeetingID       string `json:"meeting_id"`
	PatientID       string `json:"patient_id"`
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	CorrelationID   string `json:"correlation_id"`
	StartsAtRFC3339 string `json:"starts_at"`
	DurationMinutes int    `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title"`
	JoinURL         string `json:"join_url"`
	PatientEmail    string `json:"patient_email"`
	PatientPhone    string `json:"patient_phone"`
	PatientName     string `json:"patient_name"`
	StaffEmail      string `json:"staff_email"`
	StaffName       string `json:"staff_name"`
	SendSMS         bool   `json:"send_sms"`
	LiveKitRoomName string `json:"livekit_room_name,omitempty"`
	PatientToken    string `json:"patient_token,omitempty"`
	StaffToken      string `json:"staff_token,omitempty"`
}

// MeetingReminderData is published ~15 minutes before a scheduled video visit.
type MeetingReminderData struct {
	MeetingID       string `json:"meeting_id"`
	PatientID       string `json:"patient_id"`
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	CorrelationID   string `json:"correlation_id"`
	StartsAtRFC3339 string `json:"starts_at"`
	DurationMinutes int    `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title"`
	JoinURL         string `json:"join_url"`
	PatientEmail    string `json:"patient_email"`
	PatientName     string `json:"patient_name"`
	StaffEmail      string `json:"staff_email"`
	StaffName       string `json:"staff_name"`
	LiveKitRoomName string `json:"livekit_room_name,omitempty"`
	PatientToken    string `json:"patient_token,omitempty"`
	StaffToken      string `json:"staff_token,omitempty"`
}

// AppointmentBookedData is published after an appointment is booked.
type AppointmentBookedData struct {
	AppointmentID   string `json:"appointment_id"`
	PatientID       string `json:"patient_id"`
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	CorrelationID   string `json:"correlation_id"`
	StartsAtRFC3339 string `json:"starts_at"`
	DurationMinutes int    `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title"`
	Type            string `json:"type"`
	Channel         string `json:"channel"`
	JoinURL         string `json:"join_url,omitempty"`
	PatientEmail    string `json:"patient_email"`
	PatientPhone    string `json:"patient_phone"`
	PatientName     string `json:"patient_name"`
	StaffEmail      string `json:"staff_email"`
	StaffName       string `json:"staff_name"`
	StaffPhone      string `json:"staff_phone,omitempty"`
	HospitalName    string `json:"hospital_name,omitempty"`
	HospitalAddress string `json:"hospital_address,omitempty"`
	HospitalPhone   string `json:"hospital_phone,omitempty"`
	MapsURL         string `json:"maps_url,omitempty"`
	SendSMS         bool   `json:"send_sms"`
	SendEmail       bool   `json:"send_email"`
	LiveKitRoomName string `json:"livekit_room_name,omitempty"`
	PatientToken    string `json:"patient_token,omitempty"`
	StaffToken      string `json:"staff_token,omitempty"`
}

// AppointmentCancelledData is published after an appointment is cancelled.
type AppointmentCancelledData struct {
	AppointmentID   string `json:"appointment_id"`
	PatientID       string `json:"patient_id"`
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	CorrelationID   string `json:"correlation_id"`
	StartsAtRFC3339 string `json:"starts_at"`
	DurationMinutes int    `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title"`
	Type            string `json:"type"`
	Reason          string `json:"reason"`
	PatientEmail    string `json:"patient_email"`
	PatientName     string `json:"patient_name"`
	StaffEmail      string `json:"staff_email"`
	StaffName       string `json:"staff_name"`
}
