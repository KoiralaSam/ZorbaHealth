package rag

import (
	"context"
	"encoding/json"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/uuid"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
)

type ConsentChecker interface {
	CheckConsent(ctx context.Context, patientID, consentType string) (bool, string, error)
}

type AuditLogger interface {
	Append(ctx context.Context, eventType, patientID, toolName, status, failureReason string, metadata map[string]any) error
}

type Embedder = outbound.Embedder
type Summarizer = outbound.Summarizer
type Store = outbound.Store

// Pipeline orchestrates the 12-step RAG flow for patient-specific queries.
type Pipeline struct {
	store          Store
	embedder       Embedder
	summarizer     Summarizer
	consent        ConsentChecker
	audit          AuditLogger
	embeddingModel string
	serviceName    string
}

func NewPipeline(
	store Store,
	embedder Embedder,
	summarizer Summarizer,
	consent ConsentChecker,
	audit AuditLogger,
	embeddingModel string,
) *Pipeline {
	if embeddingModel == "" {
		embeddingModel = "text-embedding-3-small"
	}
	return &Pipeline{
		store:          store,
		embedder:       embedder,
		summarizer:     summarizer,
		consent:        consent,
		audit:          audit,
		embeddingModel: embeddingModel,
		serviceName:    "health-records-service",
	}
}

type QueryRequest struct {
	PatientID        string
	Query            string
	TopK             int32
	ResourceTypeHint string
	Summarize        bool
	ActorType        string
	CorrelationID    string
}

type QueryResult struct {
	Answer     string
	Citations  []models.ScoredChunk
	ChunkCount int
}

func (p *Pipeline) Run(ctx context.Context, req QueryRequest) (QueryResult, error) {
	var out QueryResult
	if strings.TrimSpace(req.PatientID) == "" {
		return out, domainErrors.ErrPatientIDRequired
	}
	if strings.TrimSpace(req.Query) == "" {
		return out, domainErrors.ErrQueryRequired
	}
	if req.TopK <= 0 {
		req.TopK = 5
	}

	pid, err := uuid.Parse(req.PatientID)
	if err != nil {
		return out, domainErrors.ErrInvalidPatientID
	}

	if p.consent != nil {
		ok, reason, err := p.consent.CheckConsent(ctx, req.PatientID, sharedaudit.ConsentHealthRecordAccess)
		if err != nil {
			_ = p.logAudit(ctx, sharedaudit.EventHealthRecordSearched, req, "error", err.Error(), nil)
			return out, err
		}
		if !ok {
			_ = p.logAudit(ctx, sharedaudit.EventHealthRecordSearched, req, "denied", reason, nil)
			return out, domainErrors.ErrConsentDenied
		}
	}

	// Step 3: touch structured FHIR store so cold-start patients still have linkage metadata.
	if _, err := p.store.ListResourcesForPatient(ctx, pid, 10); err != nil {
		return out, err
	}

	embedding, err := p.embedder.Embed(ctx, req.Query)
	if err != nil {
		return out, domainErrors.ErrEmbedQueryFailed
	}

	candidates, err := p.store.SearchRecordChunkCandidates(ctx, pid, embedding, req.TopK*3)
	if err != nil {
		return out, err
	}
	filtered := applyMetadataFilters(candidates, req.ResourceTypeHint)
	reranked := rerank(req.Query, filtered, req.TopK)

	out.Citations = reranked
	out.ChunkCount = len(reranked)
	_ = p.logAudit(ctx, sharedaudit.EventHealthRecordSearched, req, "success", "", map[string]any{
		"chunk_count": out.ChunkCount,
	})

	if !req.Summarize {
		return out, nil
	}
	if len(reranked) == 0 {
		return out, domainErrors.ErrNoRecordsFound
	}

	texts := make([]string, 0, len(reranked))
	for _, c := range reranked {
		texts = append(texts, c.Text)
	}
	answer, err := p.summarizer.AnswerQuestion(ctx, req.Query, texts)
	if err != nil {
		_ = p.logAudit(ctx, sharedaudit.EventHealthRecordSummarized, req, "error", err.Error(), nil)
		return out, err
	}
	out.Answer = answer
	_ = p.logAudit(ctx, sharedaudit.EventHealthRecordSummarized, req, "success", "", map[string]any{
		"citation_count": len(reranked),
	})
	return out, nil
}

func applyMetadataFilters(chunks []models.ScoredChunk, resourceType string) []models.ScoredChunk {
	if strings.TrimSpace(resourceType) == "" {
		return chunks
	}
	out := make([]models.ScoredChunk, 0, len(chunks))
	for _, c := range chunks {
		if c.FHIRResourceType == resourceType {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return chunks
	}
	return out
}

func rerank(query string, chunks []models.ScoredChunk, topK int32) []models.ScoredChunk {
	if len(chunks) == 0 {
		return chunks
	}
	q := strings.ToLower(query)
	for i := range chunks {
		boost := float32(0)
		if strings.Contains(strings.ToLower(chunks[i].Text), q) {
			boost += 0.05
		}
		if chunks[i].FHIRResourceType == "Condition" || chunks[i].FHIRResourceType == "MedicationRequest" {
			boost += 0.02
		}
		chunks[i].Score += boost
	}
	// insertion sort for tiny k
	for i := 1; i < len(chunks); i++ {
		j := i
		for j > 0 && chunks[j].Score > chunks[j-1].Score {
			chunks[j], chunks[j-1] = chunks[j-1], chunks[j]
			j--
		}
	}
	if int(topK) < len(chunks) {
		return chunks[:topK]
	}
	return chunks
}

func (p *Pipeline) logAudit(ctx context.Context, eventType string, req QueryRequest, status, failure string, metadata map[string]any) error {
	if p.audit == nil {
		return nil
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	if req.CorrelationID != "" {
		metadata["correlation_id"] = req.CorrelationID
	}
	metadata["embedding_model"] = p.embeddingModel
	return p.audit.Append(ctx, eventType, req.PatientID, "rag_pipeline", status, failure, metadata)
}

// GRPCAuditAdapter writes audit events through audit-service.
type GRPCAuditAdapter struct {
	client      auditpb.AuditServiceClient
	serviceName string
}

func NewGRPCAuditAdapter(client auditpb.AuditServiceClient, serviceName string) *GRPCAuditAdapter {
	if serviceName == "" {
		serviceName = "health-records-service"
	}
	return &GRPCAuditAdapter{client: client, serviceName: serviceName}
}

func (a *GRPCAuditAdapter) Append(ctx context.Context, eventType, patientID, toolName, status, failureReason string, metadata map[string]any) error {
	if a.client == nil {
		return nil
	}
	metaJSON := "{}"
	var metaStruct *structpb.Struct
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err == nil {
			metaJSON = string(b)
			metaStruct, _ = structpb.NewStruct(metadata)
		}
	}
	success := status == "success"
	_, err := a.client.AppendAuditEvent(ctx, &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventType:     eventType,
			ActorType:     "SYSTEM_SERVICE",
			PatientId:     patientID,
			ServiceName:   a.serviceName,
			ResourceType:  "HEALTH_RECORD",
			ToolName:      toolName,
			SuccessStatus: success,
			FailureReason: failureReason,
			Metadata:      metaStruct,
		},
	})
	_ = metaJSON
	return err
}
