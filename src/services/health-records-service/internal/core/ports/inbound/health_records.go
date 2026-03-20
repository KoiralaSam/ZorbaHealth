package inbound

import (
	"context"
	"encoding/json"

	models "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
)

type SearchChunk struct {
	Text       string
	SourceFile string
	Score      float32
}

type HealthRecordsService interface {
	SearchRecords(ctx context.Context, patientID, query string, topK int32) ([]models.ScoredChunk, error)
	HospitalSearchRecords(ctx context.Context, patientID, hospitalID, query string, topK int32) ([]models.ScoredChunk, error)
	SummarizeRecords(ctx context.Context, patientID, focus string) (string, error)
	IngestText(ctx context.Context, patientID, sourceFile, text string) (int32, error)
	SaveConversationTurn(ctx context.Context, patientID, sessionID, role, content string) error
	LoadRecentContext(ctx context.Context, patientID string, limit int32) (string, error)
	GetPatientResources(ctx context.Context, patientID, resourceType, status string, limit, offset int32) ([]json.RawMessage, error)
}
