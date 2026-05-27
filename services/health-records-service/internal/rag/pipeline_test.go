package rag

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
)

type stubStore struct {
	candidates []models.ScoredChunk
}

func (s *stubStore) CreateRecordChunk(context.Context, uuid.UUID, string, int32, string, []float32) error {
	return nil
}
func (s *stubStore) CreateRecordChunkDetailed(context.Context, outbound.RecordChunkInput) error { return nil }
func (s *stubStore) SearchRecordChunks(context.Context, uuid.UUID, []float32, int32) ([]models.ScoredChunk, error) {
	return s.candidates, nil
}
func (s *stubStore) SearchRecordChunksFiltered(context.Context, uuid.UUID, []float32, string, int32) ([]models.ScoredChunk, error) {
	return s.candidates, nil
}
func (s *stubStore) SearchRecordChunkCandidates(context.Context, uuid.UUID, []float32, int32) ([]models.ScoredChunk, error) {
	return s.candidates, nil
}
func (s *stubStore) HospitalSearchRecordChunks(context.Context, uuid.UUID, uuid.UUID, []float32, int32) ([]models.ScoredChunk, error) {
	return nil, nil
}
func (s *stubStore) FetchChunksForSummary(context.Context, uuid.UUID, string, int32) ([]string, error) {
	return nil, nil
}
func (s *stubStore) SaveTurn(context.Context, uuid.UUID, string, string, string, []float32) error {
	return nil
}
func (s *stubStore) LoadRecentTurns(context.Context, uuid.UUID, int32) ([]models.Turn, error) {
	return nil, nil
}
func (s *stubStore) LoadRecentTurnsBySession(context.Context, uuid.UUID, string, int32) ([]models.Turn, error) {
	return nil, nil
}
func (s *stubStore) UpsertResource(context.Context, uuid.UUID, string, string, string, json.RawMessage) error {
	return nil
}
func (s *stubStore) UpsertResourceNormalized(context.Context, uuid.UUID, string, string, string, json.RawMessage, string, string, *time.Time) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (s *stubStore) UpsertFHIRPatientMap(context.Context, string, string, uuid.UUID) error { return nil }
func (s *stubStore) ListResourcesByType(context.Context, uuid.UUID, string, int32, int32) ([]string, error) {
	return nil, nil
}
func (s *stubStore) ListResourcesByTypeAndStatus(context.Context, uuid.UUID, string, string, int32, int32) ([]string, error) {
	return nil, nil
}
func (s *stubStore) ListResourcesForPatient(context.Context, uuid.UUID, int32) ([]models.FHIRResourceSummary, error) {
	return []models.FHIRResourceSummary{{ResourceType: "Condition"}}, nil
}

type stubConsent struct{}

func (stubConsent) CheckConsent(context.Context, string, string) (bool, string, error) {
	return true, "", nil
}

type stubAudit struct {
	events []string
}

func (a *stubAudit) Append(_ context.Context, eventType, _, _, _, _ string, _ map[string]any) error {
	a.events = append(a.events, eventType)
	return nil
}

type stubEmbedder struct{}

func (stubEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{1, 0, 0}, nil
}

type stubSummarizer struct{}

func (stubSummarizer) Summarize(context.Context, []string, string) (string, error) { return "summary", nil }
func (stubSummarizer) AnswerQuestion(context.Context, string, []string) (string, error) {
	return "grounded answer", nil
}

func TestPipelineSearchAudit(t *testing.T) {
	audit := &stubAudit{}
	store := &stubStore{candidates: []models.ScoredChunk{{
		ChunkID: uuid.New(), Text: "Mild persistent asthma", Score: 0.9, FHIRResourceType: "Condition",
	}}}
	p := NewPipeline(store, stubEmbedder{}, stubSummarizer{}, stubConsent{}, audit, "text-embedding-3-small")
	result, err := p.Run(context.Background(), QueryRequest{
		PatientID: uuid.NewString(),
		Query:     "asthma",
		TopK:      3,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Citations) != 1 {
		t.Fatalf("expected one citation, got %d", len(result.Citations))
	}
	if len(audit.events) == 0 || audit.events[0] != "HEALTH_RECORD_SEARCHED" {
		t.Fatalf("unexpected audit events: %#v", audit.events)
	}
}

func TestPipelineSummarizeAudit(t *testing.T) {
	audit := &stubAudit{}
	store := &stubStore{candidates: []models.ScoredChunk{{ChunkID: uuid.New(), Text: "Albuterol inhaler", Score: 0.8}}}
	p := NewPipeline(store, stubEmbedder{}, stubSummarizer{}, stubConsent{}, audit, "text-embedding-3-small")
	result, err := p.Run(context.Background(), QueryRequest{
		PatientID: uuid.NewString(),
		Query:     "medications",
		TopK:      2,
		Summarize: true,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Answer == "" {
		t.Fatal("expected answer")
	}
	if len(audit.events) < 2 {
		t.Fatalf("expected search + summarize audit events, got %#v", audit.events)
	}
}
