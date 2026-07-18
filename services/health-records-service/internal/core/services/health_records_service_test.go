package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
)

func TestSummarizeRecordsSupportsAllFocusOptions(t *testing.T) {
	patientID := uuid.NewString()

	for _, focus := range []string{"full", "medications", "allergies", "diagnoses"} {
		t.Run(focus, func(t *testing.T) {
			store := &summaryStore{
				chunksByFocus: map[string][]string{
					focus: {focus + " chunk"},
				},
			}
			summarizer := &summarySummarizer{}
			svc := &HealthRecordsService{store: store, summarizer: summarizer}

			summary, err := svc.SummarizeRecords(context.Background(), patientID, focus)
			if err != nil {
				t.Fatalf("SummarizeRecords() error = %v", err)
			}
			if summary != "summary" {
				t.Fatalf("summary = %q, want summary", summary)
			}
			if len(store.calls) != 1 || store.calls[0] != focus {
				t.Fatalf("FetchChunksForSummary calls = %#v, want [%q]", store.calls, focus)
			}
			if summarizer.focus != focus {
				t.Fatalf("summarizer focus = %q, want %q", summarizer.focus, focus)
			}
		})
	}
}

func TestSummarizeRecordsFallsBackToFullChunksForEmptyFocusedResult(t *testing.T) {
	patientID := uuid.NewString()
	store := &summaryStore{
		chunksByFocus: map[string][]string{
			"full": {"Patient has mild asthma and takes albuterol."},
		},
	}
	summarizer := &summarySummarizer{}
	svc := &HealthRecordsService{store: store, summarizer: summarizer}

	summary, err := svc.SummarizeRecords(context.Background(), patientID, "allergies")
	if err != nil {
		t.Fatalf("SummarizeRecords() error = %v", err)
	}
	if summary != "summary" {
		t.Fatalf("summary = %q, want summary", summary)
	}
	if got, want := store.calls, []string{"allergies", "full"}; !equalStrings(got, want) {
		t.Fatalf("FetchChunksForSummary calls = %#v, want %#v", got, want)
	}
	if summarizer.focus != "allergies" {
		t.Fatalf("summarizer focus = %q, want allergies", summarizer.focus)
	}
}

func TestSummarizeRecordsNormalizesFocusAliases(t *testing.T) {
	patientID := uuid.NewString()
	store := &summaryStore{
		chunksByFocus: map[string][]string{
			"diagnoses": {"Asthma"},
		},
	}
	svc := &HealthRecordsService{store: store, summarizer: &summarySummarizer{}}

	if _, err := svc.SummarizeRecords(context.Background(), patientID, " conditions "); err != nil {
		t.Fatalf("SummarizeRecords() error = %v", err)
	}
	if len(store.calls) != 1 || store.calls[0] != "diagnoses" {
		t.Fatalf("FetchChunksForSummary calls = %#v, want [diagnoses]", store.calls)
	}
}

type summaryStore struct {
	chunksByFocus map[string][]string
	calls         []string
}

func (s *summaryStore) FetchChunksForSummary(_ context.Context, _ uuid.UUID, focus string, _ int32) ([]string, error) {
	s.calls = append(s.calls, focus)
	return s.chunksByFocus[focus], nil
}

type summarySummarizer struct {
	chunks []string
	focus  string
}

func (s *summarySummarizer) Summarize(_ context.Context, chunks []string, focus string) (string, error) {
	s.chunks = chunks
	s.focus = focus
	return "summary", nil
}

func (s *summarySummarizer) AnswerQuestion(context.Context, string, []string) (string, error) {
	return "", nil
}

func (s *summaryStore) CreateRecordChunk(context.Context, uuid.UUID, string, int32, string, []float32) error {
	return nil
}

func (s *summaryStore) CreateRecordChunkDetailed(context.Context, outbound.RecordChunkInput) error {
	return nil
}

func (s *summaryStore) SearchRecordChunks(context.Context, uuid.UUID, []float32, int32) ([]models.ScoredChunk, error) {
	return nil, nil
}

func (s *summaryStore) SearchRecordChunksFiltered(context.Context, uuid.UUID, []float32, string, int32) ([]models.ScoredChunk, error) {
	return nil, nil
}

func (s *summaryStore) SearchRecordChunkCandidates(context.Context, uuid.UUID, []float32, int32) ([]models.ScoredChunk, error) {
	return nil, nil
}

func (s *summaryStore) HospitalSearchRecordChunks(context.Context, uuid.UUID, uuid.UUID, []float32, int32) ([]models.ScoredChunk, error) {
	return nil, nil
}

func (s *summaryStore) SaveTurn(context.Context, uuid.UUID, string, string, string, []float32) error {
	return nil
}

func (s *summaryStore) LoadRecentTurns(context.Context, uuid.UUID, int32) ([]models.Turn, error) {
	return nil, nil
}

func (s *summaryStore) LoadRecentTurnsBySession(context.Context, uuid.UUID, string, int32) ([]models.Turn, error) {
	return nil, nil
}

func (s *summaryStore) UpsertResource(context.Context, uuid.UUID, string, string, string, json.RawMessage) error {
	return nil
}

func (s *summaryStore) UpsertResourceNormalized(context.Context, uuid.UUID, string, string, string, json.RawMessage, string, string, *time.Time) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (s *summaryStore) UpsertFHIRPatientMap(context.Context, string, string, uuid.UUID) error {
	return nil
}

func (s *summaryStore) ListResourcesByType(context.Context, uuid.UUID, string, int32, int32) ([]string, error) {
	return nil, nil
}

func (s *summaryStore) ListResourcesByTypeAndStatus(context.Context, uuid.UUID, string, string, int32, int32) ([]string, error) {
	return nil, nil
}

func (s *summaryStore) ListResourcesForPatient(context.Context, uuid.UUID, int32) ([]models.FHIRResourceSummary, error) {
	return nil, nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
