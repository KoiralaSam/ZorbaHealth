package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/models"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubHealthRecordsService struct {
	searchRecordsCalled      bool
	saveConversationCalled   bool
	loadRecentContextCalled  bool
	searchRecordsPatientID   string
	saveConversationPatient  string
	saveConversationSession  string
	loadRecentContextPatient string
}

func (s *stubHealthRecordsService) SearchRecords(context.Context, string, string, int32) ([]models.ScoredChunk, error) {
	s.searchRecordsCalled = true
	return nil, nil
}

func (s *stubHealthRecordsService) HospitalSearchRecords(context.Context, string, string, string, int32) ([]models.ScoredChunk, error) {
	return nil, nil
}

func (s *stubHealthRecordsService) SummarizeRecords(context.Context, string, string) (string, error) {
	return "", nil
}

func (s *stubHealthRecordsService) IngestText(context.Context, string, string, string) (int32, error) {
	return 0, nil
}

func (s *stubHealthRecordsService) SaveConversationTurn(_ context.Context, patientID, sessionID, role, content string) error {
	s.saveConversationCalled = true
	s.saveConversationPatient = patientID
	s.saveConversationSession = sessionID
	return nil
}

func (s *stubHealthRecordsService) LoadRecentContext(context.Context, string, int32) (string, error) {
	s.loadRecentContextCalled = true
	return "ctx", nil
}

func (s *stubHealthRecordsService) GetPatientResources(context.Context, string, string, string, int32, int32) ([]json.RawMessage, error) {
	return nil, nil
}

func TestNewGRPCHandlerRegisters(t *testing.T) {
	server := grpc.NewServer()
	handler := NewGRPCHandler(server, &stubHealthRecordsService{}, nil)
	if handler == nil {
		t.Fatal("expected handler")
	}
}

func TestSearchRecordsRejectsMissingClaims(t *testing.T) {
	handler := &GRPCHandler{svc: &stubHealthRecordsService{}}

	_, err := handler.SearchRecords(context.Background(), &pb.SearchRequest{
		PatientId: "patient-1",
		Query:     "asthma",
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestSearchRecordsRejectsPatientMismatch(t *testing.T) {
	stub := &stubHealthRecordsService{}
	handler := &GRPCHandler{svc: stub}
	ctx := sharedauth.WithClaims(context.Background(), &sharedauth.Claims{
		ActorType: sharedauth.ActorPatient,
		PatientID: "patient-1",
	})

	_, err := handler.SearchRecords(ctx, &pb.SearchRequest{
		PatientId: "patient-2",
		Query:     "asthma",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if stub.searchRecordsCalled {
		t.Fatal("expected service not to be called")
	}
}

func TestSaveConversationTurnRejectsSessionMismatch(t *testing.T) {
	stub := &stubHealthRecordsService{}
	handler := &GRPCHandler{svc: stub}
	ctx := sharedauth.WithClaims(context.Background(), &sharedauth.Claims{
		ActorType: sharedauth.ActorPatient,
		PatientID: "patient-1",
		SessionID: "session-1",
	})

	_, err := handler.SaveConversationTurn(ctx, &pb.SaveTurnRequest{
		PatientId: "patient-1",
		SessionId: "session-2",
		Role:      "user",
		Content:   "hello",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if stub.saveConversationCalled {
		t.Fatal("expected service not to be called")
	}
}

func TestLoadRecentContextAllowsMatchingPatient(t *testing.T) {
	stub := &stubHealthRecordsService{}
	handler := &GRPCHandler{svc: stub}
	ctx := sharedauth.WithClaims(context.Background(), &sharedauth.Claims{
		ActorType: sharedauth.ActorPatient,
		PatientID: "patient-1",
	})

	resp, err := handler.LoadRecentContext(ctx, &pb.LoadContextRequest{
		PatientId: "patient-1",
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetContextText() != "ctx" {
		t.Fatalf("unexpected context text: %q", resp.GetContextText())
	}
	if !stub.loadRecentContextCalled {
		t.Fatal("expected service to be called")
	}
}

func TestHospitalSearchRecordsRejectsHospitalMismatch(t *testing.T) {
	stub := &stubHealthRecordsService{}
	handler := &GRPCHandler{svc: stub}
	ctx := sharedauth.WithClaims(context.Background(), &sharedauth.Claims{
		ActorType:  sharedauth.ActorStaff,
		StaffID:    "staff-1",
		HospitalID: "hospital-1",
	})

	_, err := handler.HospitalSearchRecords(ctx, &pb.HospitalSearchRequest{
		PatientId:  "patient-1",
		HospitalId: "hospital-2",
		Query:      "lab",
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}
