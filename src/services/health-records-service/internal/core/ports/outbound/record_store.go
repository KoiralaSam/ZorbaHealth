package outbound

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
)

// Store is the single outbound port for persistence concerns in health-records-service.
// It combines record chunk vector search, conversation memory, and FHIR resource storage.
type Store interface {
	CreateRecordChunk(ctx context.Context, patientID uuid.UUID, sourceFile string, idx int32, text string, embedding []float32) error
	SearchRecordChunks(ctx context.Context, patientID uuid.UUID, embedding []float32, topK int32) ([]models.ScoredChunk, error)
	HospitalSearchRecordChunks(ctx context.Context, patientID, hospitalID uuid.UUID, embedding []float32, topK int32) ([]models.ScoredChunk, error)
	FetchChunksForSummary(ctx context.Context, patientID uuid.UUID, focus string, limit int32) ([]string, error)

	SaveTurn(ctx context.Context, patientID uuid.UUID, sessionID, role, content string, embedding []float32) error
	LoadRecentTurns(ctx context.Context, patientID uuid.UUID, limit int32) ([]models.Turn, error)
	LoadRecentTurnsBySession(ctx context.Context, patientID uuid.UUID, sessionID string, limit int32) ([]models.Turn, error)

	UpsertResource(ctx context.Context, patientID uuid.UUID, resourceType, resourceID, sourceSystem string, resourceJSON json.RawMessage) error
	ListResourcesByType(ctx context.Context, patientID uuid.UUID, resourceType string, limit, offset int32) ([]string, error)
	ListResourcesByTypeAndStatus(ctx context.Context, patientID uuid.UUID, resourceType, status string, limit, offset int32) ([]string, error)
}
