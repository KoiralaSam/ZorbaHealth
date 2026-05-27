package outbound

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
)

type RecordChunkInput struct {
	PatientID        uuid.UUID
	RecordID         uuid.UUID
	FHIRResourceType string
	SourceSystem     string
	SourceFile       string
	ChunkIndex       int32
	ChunkText        string
	ChunkHash        string
	AccessLevel      string
	EmbeddingModel   string
	Embedding        []float32
}

// Store is the single outbound port for persistence concerns in health-records-service.
type Store interface {
	CreateRecordChunk(ctx context.Context, patientID uuid.UUID, sourceFile string, idx int32, text string, embedding []float32) error
	CreateRecordChunkDetailed(ctx context.Context, in RecordChunkInput) error
	SearchRecordChunks(ctx context.Context, patientID uuid.UUID, embedding []float32, topK int32) ([]models.ScoredChunk, error)
	SearchRecordChunksFiltered(ctx context.Context, patientID uuid.UUID, embedding []float32, resourceType string, topK int32) ([]models.ScoredChunk, error)
	SearchRecordChunkCandidates(ctx context.Context, patientID uuid.UUID, embedding []float32, candidateLimit int32) ([]models.ScoredChunk, error)
	HospitalSearchRecordChunks(ctx context.Context, patientID, hospitalID uuid.UUID, embedding []float32, topK int32) ([]models.ScoredChunk, error)
	FetchChunksForSummary(ctx context.Context, patientID uuid.UUID, focus string, limit int32) ([]string, error)

	SaveTurn(ctx context.Context, patientID uuid.UUID, sessionID, role, content string, embedding []float32) error
	LoadRecentTurns(ctx context.Context, patientID uuid.UUID, limit int32) ([]models.Turn, error)
	LoadRecentTurnsBySession(ctx context.Context, patientID uuid.UUID, sessionID string, limit int32) ([]models.Turn, error)

	UpsertResource(ctx context.Context, patientID uuid.UUID, resourceType, resourceID, sourceSystem string, resourceJSON json.RawMessage) error
	UpsertResourceNormalized(
		ctx context.Context,
		patientID uuid.UUID,
		resourceType, resourceID, sourceSystem string,
		resourceJSON json.RawMessage,
		displayText, clinicalStatus string,
		effectiveDate *time.Time,
	) (uuid.UUID, error)
	UpsertFHIRPatientMap(ctx context.Context, fhirPatientID, sourceSystem string, internalPatientID uuid.UUID) error
	ListResourcesByType(ctx context.Context, patientID uuid.UUID, resourceType string, limit, offset int32) ([]string, error)
	ListResourcesByTypeAndStatus(ctx context.Context, patientID uuid.UUID, resourceType, status string, limit, offset int32) ([]string, error)
	ListResourcesForPatient(ctx context.Context, patientID uuid.UUID, limit int32) ([]models.FHIRResourceSummary, error)
}
