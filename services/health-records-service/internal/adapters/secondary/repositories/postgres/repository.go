package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/adapters/secondary/repositories/postgres/sqlc"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
)

// Repository is the Postgres adapter for medical-records-service.
// It wraps sqlc-generated queries for vector search, conversation memory, and FHIR resources.
type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) outbound.Store {
	return &Repository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// ---- record_chunks (vector search) ----
func (r *Repository) CreateRecordChunk(ctx context.Context, patientID uuid.UUID, sourceFile string, idx int32, text string, embedding []float32) error {
	_, err := r.queries.CreateRecordChunk(ctx, sqlc.CreateRecordChunkParams{
		PatientID:  pgtype.UUID{Bytes: patientID, Valid: true},
		SourceFile: sourceFile,
		ChunkIndex: idx,
		ChunkText:  text,
		Column5:    pgvector.NewVector(embedding),
	})
	return err
}

func (r *Repository) SearchRecordChunks(ctx context.Context, patientID uuid.UUID, embedding []float32, topK int32) ([]models.ScoredChunk, error) {
	rows, err := r.queries.SearchRecordChunksByEmbedding(ctx, sqlc.SearchRecordChunksByEmbeddingParams{
		PatientID: pgtype.UUID{Bytes: patientID, Valid: true},
		Column2:   pgvector.NewVector(embedding),
		Limit:     topK,
	})
	if err != nil {
		return nil, err
	}

	out := make([]models.ScoredChunk, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.ScoredChunk{
			Text:       row.ChunkText,
			SourceFile: row.SourceFile,
			Score:      row.Score,
		})
	}
	return out, nil
}

func (r *Repository) HospitalSearchRecordChunks(ctx context.Context, patientID, hospitalID uuid.UUID, embedding []float32, topK int32) ([]models.ScoredChunk, error) {
	rows, err := r.queries.HospitalSearchRecordChunksByEmbedding(ctx, sqlc.HospitalSearchRecordChunksByEmbeddingParams{
		PatientID:  pgtype.UUID{Bytes: patientID, Valid: true},
		HospitalID: pgtype.UUID{Bytes: hospitalID, Valid: true},
		Column3:    pgvector.NewVector(embedding),
		Limit:      topK,
	})
	if err != nil {
		return nil, err
	}

	out := make([]models.ScoredChunk, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.ScoredChunk{
			Text:       row.ChunkText,
			SourceFile: row.SourceFile,
			Score:      row.Score,
		})
	}
	return out, nil
}

func (r *Repository) FetchChunksForSummary(ctx context.Context, patientID uuid.UUID, focus string, limit int32) ([]string, error) {
	return r.queries.FetchChunksForSummary(ctx, sqlc.FetchChunksForSummaryParams{
		PatientID: pgtype.UUID{Bytes: patientID, Valid: true},
		Column2:   focus,
		Limit:     limit,
	})
}

// ---- conversation_turns (memory) ----

func (r *Repository) SaveTurn(ctx context.Context, patientID uuid.UUID, sessionID, role, content string, embedding []float32) error {
	_, err := r.queries.CreateConversationTurn(ctx, sqlc.CreateConversationTurnParams{
		PatientID: pgtype.UUID{Bytes: patientID, Valid: true},
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Column5:   pgvector.NewVector(embedding),
	})
	return err
}

func (r *Repository) LoadRecentTurns(ctx context.Context, patientID uuid.UUID, limit int32) ([]models.Turn, error) {
	rows, err := r.queries.ListRecentConversationTurns(ctx, sqlc.ListRecentConversationTurnsParams{
		PatientID: pgtype.UUID{Bytes: patientID, Valid: true},
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]models.Turn, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.Turn{
			Role:    row.Role,
			Content: row.Content,
		})
	}
	return out, nil
}

func (r *Repository) LoadRecentTurnsBySession(ctx context.Context, patientID uuid.UUID, sessionID string, limit int32) ([]models.Turn, error) {
	rows, err := r.queries.ListRecentConversationTurnsBySession(ctx, sqlc.ListRecentConversationTurnsBySessionParams{
		PatientID: pgtype.UUID{Bytes: patientID, Valid: true},
		SessionID: sessionID,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]models.Turn, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.Turn{
			Role:    row.Role,
			Content: row.Content,
		})
	}
	return out, nil
}

// ---- fhir_resources (JSONB) ----

func (r *Repository) UpsertResource(ctx context.Context, patientID uuid.UUID, resourceType, resourceID, sourceSystem string, resourceJSON json.RawMessage) error {
	_, err := r.queries.UpsertFHIRResource(ctx, sqlc.UpsertFHIRResourceParams{
		PatientID:    pgtype.UUID{Bytes: patientID, Valid: true},
		ResourceType: resourceType,
		ResourceID:   resourceID,
		SourceSystem: pgtype.Text{String: sourceSystem, Valid: sourceSystem != ""},
		Column5:      []byte(resourceJSON),
	})
	return err
}

func (r *Repository) ListResourcesByType(ctx context.Context, patientID uuid.UUID, resourceType string, limit, offset int32) ([]string, error) {
	return r.queries.ListFHIRResourcesByType(ctx, sqlc.ListFHIRResourcesByTypeParams{
		PatientID:    pgtype.UUID{Bytes: patientID, Valid: true},
		ResourceType: resourceType,
		Limit:        limit,
		Offset:       offset,
	})
}

func (r *Repository) ListResourcesByTypeAndStatus(ctx context.Context, patientID uuid.UUID, resourceType, status string, limit, offset int32) ([]string, error) {
	return r.queries.ListFHIRResourcesByTypeAndStatus(ctx, sqlc.ListFHIRResourcesByTypeAndStatusParams{
		PatientID:    pgtype.UUID{Bytes: patientID, Valid: true},
		ResourceType: resourceType,
		Column3:      status,
		Limit:        limit,
		Offset:       offset,
	})
}
