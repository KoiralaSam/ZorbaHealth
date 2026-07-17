package audit

const (
	EventPatientCreated               = "PATIENT_CREATED"
	EventPatientVerified              = "PATIENT_VERIFIED"
	EventPatientLogin                 = "PATIENT_LOGIN"
	EventPatientLogout                = "PATIENT_LOGOUT"
	EventHealthRecordCreated          = "HEALTH_RECORD_CREATED"
	EventHealthRecordViewed           = "HEALTH_RECORD_VIEWED"
	EventHealthRecordSearched         = "HEALTH_RECORD_SEARCHED"
	EventHealthRecordSummarized       = "HEALTH_RECORD_SUMMARIZED"
	EventAIToolCalled                 = "AI_TOOL_CALLED"
	EventAIResponseGenerated          = "AI_RESPONSE_GENERATED"
	EventLocationRequested            = "LOCATION_REQUESTED"
	EventEmergencyEscalationTriggered = "EMERGENCY_ESCALATION_TRIGGERED"
	EventNotificationSent             = "NOTIFICATION_SENT"
	EventConsentGranted               = "CONSENT_GRANTED"
	EventConsentRevoked               = "CONSENT_REVOKED"
	EventTranslationRequested         = "TRANSLATION_REQUESTED"
	EventWelfareCheckRequested        = "WELFARE_CHECK_REQUESTED"
	EventWelfareCheckCancelled        = "WELFARE_CHECK_CANCELLED"
	EventWelfareCheckDispatched       = "WELFARE_CHECK_DISPATCHED"
	EventWelfareCheckAnswered         = "WELFARE_CHECK_ANSWERED"
	EventWelfareCheckMissed           = "WELFARE_CHECK_MISSED"
	EventWelfareCheckFailed           = "WELFARE_CHECK_FAILED"
	EventWelfareCheckCompleted        = "WELFARE_CHECK_COMPLETED"
	EventWelfareCheckRecordAccessed   = "WELFARE_CHECK_RECORD_CONTEXT_ACCESSED"
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
	EventWelfareCheckRequested,
	EventWelfareCheckCancelled,
	EventWelfareCheckDispatched,
	EventWelfareCheckAnswered,
	EventWelfareCheckMissed,
	EventWelfareCheckFailed,
	EventWelfareCheckCompleted,
	EventWelfareCheckRecordAccessed,
}
