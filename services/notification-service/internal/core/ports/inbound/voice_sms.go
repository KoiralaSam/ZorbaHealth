package inbound

import "context"

// VoiceSMSProcessor forwards inbound SMS to patient-service for voice OTP handling.
type VoiceSMSProcessor interface {
	ProcessInboundVoiceSms(ctx context.Context, fromPhone, messageBody string) (processed bool, reason string, err error)
}
