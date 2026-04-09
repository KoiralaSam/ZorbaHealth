package healthrecords

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
)

type Client struct {
	grpc healthpb.HealthRecordServiceClient
}

var _ outbound.HealthRecordsClient = (*Client)(nil)

func NewClient(grpc healthpb.HealthRecordServiceClient) *Client {
	return &Client{grpc: grpc}
}

func (c *Client) LoadRecentContext(ctx context.Context, patientID string, limit int32) (string, error) {
	resp, err := c.grpc.LoadRecentContext(ctx, &healthpb.LoadContextRequest{
		PatientId: patientID,
		Limit:     limit,
	})
	if err != nil {
		return "", err
	}
	return resp.GetContextText(), nil
}

func (c *Client) SaveConversationTurn(ctx context.Context, patientID, sessionID, role, content string) error {
	_, err := c.grpc.SaveConversationTurn(ctx, &healthpb.SaveTurnRequest{
		PatientId: patientID,
		SessionId: sessionID,
		Role:      role,
		Content:   content,
	})
	return err
}
