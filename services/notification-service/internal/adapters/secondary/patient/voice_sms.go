package patient

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	regpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
)

type VoiceSMSClient struct {
	client regpb.RegistrationVerificationServiceClient
}

func NewVoiceSMSClient(addr string) (*VoiceSMSClient, error) {
	conn, err := grpcclient.Dial(addr)
	if err != nil {
		return nil, err
	}
	return &VoiceSMSClient{client: regpb.NewRegistrationVerificationServiceClient(conn)}, nil
}

func (c *VoiceSMSClient) ProcessInboundVoiceSms(ctx context.Context, fromPhone, messageBody string) (processed bool, reason, voiceSessionID, correlationID string, err error) {
	resp, err := c.client.ProcessInboundVoiceSms(ctx, &regpb.ProcessInboundVoiceSmsRequest{
		FromPhone:   fromPhone,
		MessageBody: messageBody,
	})
	if err != nil {
		return false, "", "", "", err
	}
	return resp.GetProcessed(), resp.GetReason(), resp.GetVoiceSessionId(), resp.GetCorrelationId(), nil
}
