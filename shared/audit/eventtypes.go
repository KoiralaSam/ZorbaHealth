package audit

const (
	EventPatientCreated              = "PATIENT_CREATED"
	EventPatientVerified             = "PATIENT_VERIFIED"
	EventPatientLogin                = "PATIENT_LOGIN"
	EventPatientLogout               = "PATIENT_LOGOUT"
	EventHealthRecordCreated         = "HEALTH_RECORD_CREATED"
	EventHealthRecordViewed          = "HEALTH_RECORD_VIEWED"
	EventHealthRecordSearched        = "HEALTH_RECORD_SEARCHED"
	EventHealthRecordSummarized      = "HEALTH_RECORD_SUMMARIZED"
	EventAIToolCalled                = "AI_TOOL_CALLED"
	EventAIResponseGenerated         = "AI_RESPONSE_GENERATED"
	EventLocationRequested           = "LOCATION_REQUESTED"
	EventEmergencyEscalationTriggered = "EMERGENCY_ESCALATION_TRIGGERED"
	EventNotificationSent            = "NOTIFICATION_SENT"
	EventConsentGranted             = "CONSENT_GRANTED"
	EventConsentRevoked             = "CONSENT_REVOKED"
	EventTranslationRequested       = "TRANSLATION_REQUESTED"
)

var AllEventTypes = []string{
	EventPatientCreated,
	EventPatientVerified,
	EventPatientLogin,
	EventPatientLogout,
	EventHealthRecordCreated,
	EventHealthRecordViewed,
	EventHealthRecordSearched,
	EventHealthRecordSummarized,
	EventAIToolCalled,
	EventAIResponseGenerated,
	EventLocationRequested,
	EventEmergencyEscalationTriggered,
	EventNotificationSent,
	EventConsentGranted,
	EventConsentRevoked,
	EventTranslationRequested,
}
