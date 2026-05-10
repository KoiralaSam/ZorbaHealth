package outbound

import "context"

type HealthRecordsClient interface {
	LoadRecentContext(ctx context.Context, patientID string, limit int32) (string, error)
	SaveConversationTurn(ctx context.Context, patientID, sessionID, role, content string) error
}
