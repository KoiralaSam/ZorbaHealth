package contracts

// AmqpMessage is the message structure for AMQP.
type AmqpMessage struct {
	OwnerID string `json:"ownerId"`
	Data    []byte `json:"data"`
}

// Routing keys - using consistent event/command patterns
const (
	// Session events (session.event.*)
	SessionEventStarted = "session.event.started"
	SessionEventEnded   = "session.event.ended"
	SessionEventFailed  = "session.event.failed"

	// User interaction events (user.event.*)
	UserEventSpoke = "user.event.spoke"

	// Assistant interaction events (assistant.event.*)
	AssistantEventResponded = "assistant.event.responded"

	// Patient events (patient.event.*)
	PatientEventRegistered                = "patient.event.registered"
	PatientEventChached                   = "patient.event.chached"
	PatientEventNotRegistered             = "patient.event.not_registered"
	PatientEventVerificationCodeRequested = "patient.event.verification_code_requested"
	PatientEventUpdated                   = "patient.event.updated"

	// Medical Records events (medical_record.event.*)
	MedicalRecordEventCreated = "medical_record.event.created"
	CallTranscriptEventSaved  = "call_transcript.event.saved"

	// Call lifecycle events used by location-service (call.*)
	CallEventStarted = "call.started"
	CallEventEnded   = "call.ended"
	CallEventFailed  = "call.failed"
	CallEventAll     = "call.*"

	// Commands for services
	// Session commands (session.cmd.*)
	SessionCmdStart = "session.cmd.start"
	SessionCmdEnd   = "session.cmd.end"

	// Patient commands (patient.cmd.*)
	PatientCmdRegister = "patient.cmd.register"
	PatientCmdUpdate   = "patient.cmd.update"
)
