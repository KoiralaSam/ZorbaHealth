package audit

const (
	EventPatientCreated                   = "PATIENT_CREATED"
	EventPatientVerified                  = "PATIENT_VERIFIED"
	EventPatientLogin                     = "PATIENT_LOGIN"
	EventPatientLogout                    = "PATIENT_LOGOUT"
	EventPatientSessionRefresh            = "PATIENT_SESSION_REFRESH"
	EventStaffLogin                       = "STAFF_LOGIN"
	EventStaffLogout                      = "STAFF_LOGOUT"
	EventStaffSessionRefresh              = "STAFF_SESSION_REFRESH"
	EventSessionReuseDetected             = "SESSION_REUSE_DETECTED"
	EventHealthRecordCreated              = "HEALTH_RECORD_CREATED"
	EventHealthRecordViewed               = "HEALTH_RECORD_VIEWED"
	EventHealthRecordSearched             = "HEALTH_RECORD_SEARCHED"
	EventHealthRecordSummarized           = "HEALTH_RECORD_SUMMARIZED"
	EventAIToolCalled                     = "AI_TOOL_CALLED"
	EventAIResponseGenerated              = "AI_RESPONSE_GENERATED"
	EventLocationRequested                = "LOCATION_REQUESTED"
	EventEmergencyEscalationTriggered     = "EMERGENCY_ESCALATION_TRIGGERED"
	EventNotificationSent                 = "NOTIFICATION_SENT"
	EventConsentGranted                   = "CONSENT_GRANTED"
	EventConsentRevoked                   = "CONSENT_REVOKED"
	EventTranslationRequested             = "TRANSLATION_REQUESTED"
	EventVoiceOTPWaitRegistered           = "VOICE_OTP_WAIT_REGISTERED"
	EventVoiceInboundSMSProcessed         = "VOICE_INBOUND_SMS_PROCESSED"
	EventVoiceOTPVerifyFailed             = "VOICE_OTP_VERIFY_FAILED"
	EventMeetingRequested                 = "MEETING_REQUESTED"
	EventMeetingAccepted                  = "MEETING_ACCEPTED"
	EventMeetingRescheduled               = "MEETING_RESCHEDULED"
	EventMeetingScheduled                 = "MEETING_SCHEDULED"
	EventMeetingCancelled                 = "MEETING_CANCELLED"
	EventMeetingScheduleDenied            = "MEETING_SCHEDULE_DENIED"
	EventCallTransferRequested            = "CALL_TRANSFER_REQUESTED"
	EventCallTransferConnected            = "CALL_TRANSFER_CONNECTED"
	EventCallBridgedEnded                 = "CALL_BRIDGED_ENDED"
	EventInterpretationSessionStarted     = "INTERPRETATION_SESSION_STARTED"
	EventInterpretationSessionEnded       = "INTERPRETATION_SESSION_ENDED"
	EventInterpretationPreferencesUpdated = "INTERPRETATION_PREFERENCES_UPDATED"
	EventInterpretationSegmentProcessed   = "INTERPRETATION_SEGMENT_PROCESSED"
	EventWelfareCheckRequested            = "WELFARE_CHECK_REQUESTED"
	EventWelfareCheckCancelled            = "WELFARE_CHECK_CANCELLED"
	EventWelfareCheckDispatched           = "WELFARE_CHECK_DISPATCHED"
	EventWelfareCheckAnswered             = "WELFARE_CHECK_ANSWERED"
	EventWelfareCheckMissed               = "WELFARE_CHECK_MISSED"
	EventWelfareCheckFailed               = "WELFARE_CHECK_FAILED"
	EventWelfareCheckCompleted            = "WELFARE_CHECK_COMPLETED"
	EventWelfareCheckRecordAccessed       = "WELFARE_CHECK_RECORD_CONTEXT_ACCESSED"
)

var AllEventTypes = []string{
	EventPatientCreated,
	EventPatientVerified,
	EventPatientLogin,
	EventPatientLogout,
	EventPatientSessionRefresh,
	EventStaffLogin,
	EventStaffLogout,
	EventStaffSessionRefresh,
	EventSessionReuseDetected,
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
	EventVoiceOTPWaitRegistered,
	EventVoiceInboundSMSProcessed,
	EventVoiceOTPVerifyFailed,
	EventMeetingRequested,
	EventMeetingAccepted,
	EventMeetingRescheduled,
	EventMeetingScheduled,
	EventMeetingCancelled,
	EventMeetingScheduleDenied,
	EventCallTransferRequested,
	EventCallTransferConnected,
	EventCallBridgedEnded,
	EventInterpretationSessionStarted,
	EventInterpretationSessionEnded,
	EventInterpretationPreferencesUpdated,
	EventInterpretationSegmentProcessed,
	EventWelfareCheckRequested,
	EventWelfareCheckCancelled,
	EventWelfareCheckDispatched,
	EventWelfareCheckAnswered,
	EventWelfareCheckMissed,
	EventWelfareCheckFailed,
	EventWelfareCheckCompleted,
	EventWelfareCheckRecordAccessed,
}
