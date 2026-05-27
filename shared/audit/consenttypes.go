package audit

const (
	ConsentVoiceAssistantUse      = "VOICE_ASSISTANT_USE"
	ConsentHealthRecordAccess     = "HEALTH_RECORD_ACCESS"
	ConsentLocationAccess         = "LOCATION_ACCESS"
	ConsentSMSNotification        = "SMS_NOTIFICATION"
	ConsentEmailNotification      = "EMAIL_NOTIFICATION"
	ConsentAISummarization        = "AI_SUMMARIZATION"
	ConsentThirdPartyProcessing   = "THIRD_PARTY_MODEL_PROCESSING"
)

var AllConsentTypes = []string{
	ConsentVoiceAssistantUse,
	ConsentHealthRecordAccess,
	ConsentLocationAccess,
	ConsentSMSNotification,
	ConsentEmailNotification,
	ConsentAISummarization,
	ConsentThirdPartyProcessing,
}
