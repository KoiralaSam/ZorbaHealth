package events

import "github.com/KoiralaSam/ZorbaHealth/shared/contracts"

const (
	// PatientExchange is the topic exchange for patient-related events (publish + consume).
	PatientExchange = "patient"

	NotifyPatientRegisteredQueue          = "notify_patient_registered"
	NotifyPatientPendingVerificationQueue = "notify_patient_pending_verification"

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
