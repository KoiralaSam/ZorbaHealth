package models

type InboundVoiceSmsResult struct {
	Processed      bool
	Reason         string
	VoiceSessionID string
	CorrelationID  string
}
