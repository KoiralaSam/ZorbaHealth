package patient

import "context"

type Processor struct {
	client *VoiceSMSClient
}

func NewProcessor(client *VoiceSMSClient) *Processor {
	return &Processor{client: client}
}

func (p *Processor) ProcessInboundVoiceSms(ctx context.Context, fromPhone, messageBody string) (bool, string, error) {
	if p == nil || p.client == nil {
		return false, "voice_sms_disabled", nil
	}
	processed, reason, _, _, err := p.client.ProcessInboundVoiceSms(ctx, fromPhone, messageBody)
	return processed, reason, err
}
