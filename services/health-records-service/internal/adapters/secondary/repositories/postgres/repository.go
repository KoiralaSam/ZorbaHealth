package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/adapters/secondary/repositories/postgres/sqlc"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/rag/chunker"
)

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

func (r *Repository) CreateRecordChunk(ctx context.Context, patientID uuid.UUID, sourceFile string, idx int32, text string, embedding []float32) error {
	return r.CreateRecordChunkDetailed(ctx, outbound.RecordChunkInput{
		PatientID:      patientID,
		SourceFile:     sourceFile,
		ChunkIndex:     idx,
		ChunkText:      text,
		ChunkHash:      chunker.Hash(text),
		AccessLevel:    "patient",
		EmbeddingModel: "text-embedding-3-small",
		Embedding:      embedding,
	})
}

func (r *Repository) CreateRecordChunkDetailed(ctx context.Context, in outbound.RecordChunkInput) error {
	_, err := r.queries.CreateRecordChunk(ctx, sqlc.CreateRecordChunkParams{
		PatientID:        pgtype.UUID{Bytes: in.PatientID, Valid: true},
		RecordID:         uuidToPg(in.RecordID),
		FhirResourceType: pgtype.Text{String: in.FHIRResourceType, Valid: in.FHIRResourceType != ""},
		SourceSystem:     pgtype.Text{String: in.SourceSystem, Valid: in.SourceSystem != ""},
		SourceFile:       in.SourceFile,
		ChunkIndex:       in.ChunkIndex,
		ChunkText:        in.ChunkText,
		ChunkHash:        in.ChunkHash,
		AccessLevel:      in.AccessLevel,
		EmbeddingModel:   in.EmbeddingModel,
		Column11:         pgvector.NewVector(in.Embedding),
	})
	return err
}

func (r *Repository) SearchRecordChunks(ctx context.Context, patientID uuid.UUID, embedding []float32, topK int32) ([]models.ScoredChunk, error) {
	return r.SearchRecordChunksFiltered(ctx, patientID, embedding, "", topK)
}

func (r *Repository) SearchRecordChunksFiltered(ctx context.Context, patientID uuid.UUID, embedding []float32, resourceType string, topK int32) ([]models.ScoredChunk, error) {
	rows, err := r.queries.SearchRecordChunksByEmbedding(ctx, sqlc.SearchRecordChunksByEmbeddingParams{
		PatientID: pgtype.UUID{Bytes: patientID, Valid: true},
		Column2:   pgvector.NewVector(embedding),
		Limit:     topK,
		Column4:   resourceType,
	})
	if err != nil {
		return nil, err
	}
	return mapScoredRows(rows), nil
}

func (r *Repository) SearchRecordChunkCandidates(ctx context.Context, patientID uuid.UUID, embedding []float32, candidateLimit int32) ([]models.ScoredChunk, error) {
	rows, err := r.queries.SearchRecordChunksCandidates(ctx, sqlc.SearchRecordChunksCandidatesParams{
		PatientID: pgtype.UUID{Bytes: patientID, Valid: true},
		Column2:   pgvector.NewVector(embedding),
		Limit:     candidateLimit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]models.ScoredChunk, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.ScoredChunk{
			ChunkID:          pgToUUID(row.ChunkID),
			RecordID:         pgToUUID(row.RecordID),
			FHIRResourceType: row.FhirResourceType.String,
			Text:             row.ChunkText,
			SourceFile:       row.SourceFile,
			Score:            row.Score,
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
			ChunkID:          pgToUUID(row.ChunkID),
			RecordID:         pgToUUID(row.RecordID),
			FHIRResourceType: row.FhirResourceType.String,
			Text:             row.ChunkText,
			SourceFile:       row.SourceFile,
			Score:            row.Score,
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
		out = append(out, models.Turn{Role: row.Role, Content: row.Content})
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
		out = append(out, models.Turn{Role: row.Role, Content: row.Content})
	}
	return out, nil
}

func (r *Repository) UpsertResource(ctx context.Context, patientID uuid.UUID, resourceType, resourceID, sourceSystem string, resourceJSON json.RawMessage) error {
	_, err := r.UpsertResourceNormalized(ctx, patientID, resourceType, resourceID, sourceSystem, resourceJSON, "", "", nil)
	return err
}

func (r *Repository) UpsertResourceNormalized(
	ctx context.Context,
	patientID uuid.UUID,
	resourceType, resourceID, sourceSystem string,
	resourceJSON json.RawMessage,
	displayText, clinicalStatus string,
	effectiveDate *time.Time,
) (uuid.UUID, error) {
	row, err := r.queries.UpsertFHIRResource(ctx, sqlc.UpsertFHIRResourceParams{
		PatientID:      pgtype.UUID{Bytes: patientID, Valid: true},
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		SourceSystem:   pgtype.Text{String: sourceSystem, Valid: sourceSystem != ""},
		Column5:        []byte(resourceJSON),
		DisplayText:    pgtype.Text{String: displayText, Valid: displayText != ""},
		ClinicalStatus: pgtype.Text{String: clinicalStatus, Valid: clinicalStatus != ""},
		EffectiveDate:  timeToPg(effectiveDate),
	})
	if err != nil {
		return uuid.Nil, err
	}
	return pgToUUID(row.ID), nil
}

func (r *Repository) UpsertFHIRPatientMap(ctx context.Context, fhirPatientID, sourceSystem string, internalPatientID uuid.UUID) error {
	return r.queries.UpsertFHIRPatientMap(ctx, sqlc.UpsertFHIRPatientMapParams{
		FhirPatientID:     fhirPatientID,
		SourceSystem:      sourceSystem,
		InternalPatientID: pgtype.UUID{Bytes: internalPatientID, Valid: true},
	})
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

func (r *Repository) ListResourcesForPatient(ctx context.Context, patientID uuid.UUID, limit int32) ([]models.FHIRResourceSummary, error) {
	rows, err := r.queries.ListFHIRResourcesForPatient(ctx, sqlc.ListFHIRResourcesForPatientParams{
		PatientID: pgtype.UUID{Bytes: patientID, Valid: true},
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]models.FHIRResourceSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.FHIRResourceSummary{
			ID:           pgToUUID(row.ID),
			ResourceType: row.ResourceType,
			ResourceID:   row.ResourceID,
			DisplayText:  row.DisplayText.String,
			Status:       row.ClinicalStatus.String,
		})
	}
	return out, nil
}

func mapScoredRows(rows []sqlc.SearchRecordChunksByEmbeddingRow) []models.ScoredChunk {
	out := make([]models.ScoredChunk, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.ScoredChunk{
			ChunkID:          pgToUUID(row.ChunkID),
			RecordID:         pgToUUID(row.RecordID),
			FHIRResourceType: row.FhirResourceType.String,
			Text:             row.ChunkText,
			SourceFile:       row.SourceFile,
			Score:            row.Score,
		})
	}
	return out
}

func pgToUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func uuidToPg(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

func timeToPg(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
