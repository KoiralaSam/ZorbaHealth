package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
)

// HealthRecordsService implements inbound.HealthRecordsService using outbound ports:
// - Embedder: OpenAI embeddings
// - Store: pgvector record_chunks, conversation_turns, fhir_resources
// - Summarizer: OpenAI (or other LLM) summarization
type HealthRecordsService struct {
	embedder   outbound.Embedder
	store      outbound.Store
	summarizer outbound.Summarizer
}

func NewHealthRecordsService(
	embedder outbound.Embedder,
	store outbound.Store,
	summarizer outbound.Summarizer,
) inbound.HealthRecordsService {
	return &HealthRecordsService{
		embedder:   embedder,
		store:      store,
		summarizer: summarizer,
	}
}

func (s *HealthRecordsService) SearchRecords(ctx context.Context, patientID, query string, topK int32) ([]models.ScoredChunk, error) {
	if strings.TrimSpace(patientID) == "" {
		return nil, fmt.Errorf("patient_id required")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query required")
	}
	if topK <= 0 {
		topK = 5
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return nil, fmt.Errorf("invalid patient_id: %w", err)
	}

	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	return s.store.SearchRecordChunks(ctx, pid, embedding, topK)
}

func (s *HealthRecordsService) HospitalSearchRecords(ctx context.Context, patientID, hospitalID, query string, topK int32) ([]models.ScoredChunk, error) {
	if strings.TrimSpace(patientID) == "" {
		return nil, fmt.Errorf("patient_id required")
	}
	if strings.TrimSpace(hospitalID) == "" {
		return nil, fmt.Errorf("hospital_id required")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query required")
	}
	if topK <= 0 {
		topK = 5
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return nil, fmt.Errorf("invalid patient_id: %w", err)
	}
	hid, err := uuid.Parse(hospitalID)
	if err != nil {
		return nil, fmt.Errorf("invalid hospital_id: %w", err)
	}

	embedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	return s.store.HospitalSearchRecordChunks(ctx, pid, hid, embedding, topK)
}

func (s *HealthRecordsService) SummarizeRecords(ctx context.Context, patientID, focus string) (string, error) {
	if strings.TrimSpace(patientID) == "" {
		return "", fmt.Errorf("patient_id required")
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return "", fmt.Errorf("invalid patient_id: %w", err)
	}

	// Keep in sync with the existing adapter query/limit.
	chunks, err := s.store.FetchChunksForSummary(ctx, pid, focus, 50)
	if err != nil {
		return "", fmt.Errorf("fetch chunks: %w", err)
	}
	if len(chunks) == 0 {
		return "", fmt.Errorf("no records found")
	}

	return s.summarizer.Summarize(ctx, chunks, focus)
}

func (s *HealthRecordsService) IngestText(ctx context.Context, patientID, sourceFile, text string) (int32, error) {
	if strings.TrimSpace(patientID) == "" {
		return 0, fmt.Errorf("patient_id required")
	}
	if strings.TrimSpace(sourceFile) == "" {
		return 0, fmt.Errorf("source_file required")
	}
	if strings.TrimSpace(text) == "" {
		return 0, fmt.Errorf("text required")
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return 0, fmt.Errorf("invalid patient_id: %w", err)
	}

	// Word-based chunking is sufficient here; embeddings will enforce semantic boundaries.
	chunks := chunkText(text, 500, 50) // 500-token-ish windows, 50-token overlap (approx by words)
	var stored int32

	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		embedding, err := s.embedder.Embed(ctx, chunk)
		if err != nil {
			return stored, fmt.Errorf("embed chunk %d: %w", i, err)
		}

		if err := s.store.CreateRecordChunk(ctx, pid, sourceFile, int32(i), chunk, embedding); err != nil {
			return stored, fmt.Errorf("store chunk %d: %w", i, err)
		}
		stored++
	}

	return stored, nil
}

func (s *HealthRecordsService) SaveConversationTurn(ctx context.Context, patientID, sessionID, role, content string) error {
	if strings.TrimSpace(patientID) == "" {
		return fmt.Errorf("patient_id required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session_id required")
	}
	if strings.TrimSpace(role) == "" {
		return fmt.Errorf("role required")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content required")
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	embedding, err := s.embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}

	return s.store.SaveTurn(ctx, pid, sessionID, role, content, embedding)
}

func (s *HealthRecordsService) LoadRecentContext(ctx context.Context, patientID string, limit int32) (string, error) {
	if strings.TrimSpace(patientID) == "" {
		return "", fmt.Errorf("patient_id required")
	}
	if limit <= 0 {
		limit = 10
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return "", fmt.Errorf("invalid patient_id: %w", err)
	}

	turns, err := s.store.LoadRecentTurns(ctx, pid, limit)
	if err != nil {
		return "", fmt.Errorf("load turns: %w", err)
	}

	// Create a simple plain-text context that your agent-worker can inject.
	var sb strings.Builder
	if len(turns) > 0 {
		sb.WriteString("Previous conversation:\n")
		for _, t := range turns {
			sb.WriteString(fmt.Sprintf("%s: %s\n", t.Role, t.Content))
		}
	}
	return sb.String(), nil
}

func (s *HealthRecordsService) GetPatientResources(
	ctx context.Context,
	patientID,
	resourceType,
	status string,
	limit,
	offset int32,
) ([]json.RawMessage, error) {
	if strings.TrimSpace(patientID) == "" {
		return nil, fmt.Errorf("patient_id required")
	}
	if strings.TrimSpace(resourceType) == "" {
		return nil, fmt.Errorf("resource_type required")
	}
	if limit <= 0 {
		limit = 50
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return nil, fmt.Errorf("invalid patient_id: %w", err)
	}

	resources, err := s.store.ListResourcesByTypeAndStatus(ctx, pid, resourceType, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}

	out := make([]json.RawMessage, 0, len(resources))
	for _, r := range resources {
		out = append(out, json.RawMessage([]byte(r)))
	}
	return out, nil
}

// chunkText splits text into overlapping windows by word count.
// This matches the approach described in MCP_AGENT_INTEGRATION_README.md (approx token windows).
func chunkText(text string, chunkSize, overlap int) []string {
	words := strings.Fields(text)
	if chunkSize <= overlap {
		// Prevent infinite loop / negative step.
		chunkSize = overlap + 1
	}

	var chunks []string
	for i := 0; i < len(words); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks
}
