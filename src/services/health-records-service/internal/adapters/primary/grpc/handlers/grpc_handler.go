package handlers

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/inbound"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler is the primary gRPC adapter for health-records-service.
// It maps protobuf requests/responses to the inbound core service.
type GRPCHandler struct {
	pb.UnimplementedHealthRecordServiceServer
	svc inbound.HealthRecordsService
	db  *pgxpool.Pool
}

func NewGRPCHandler(server *grpc.Server, svc inbound.HealthRecordsService, db *pgxpool.Pool) *GRPCHandler {
	h := &GRPCHandler{svc: svc, db: db}
	pb.RegisterHealthRecordServiceServer(server, h)
	return h
}

func (h *GRPCHandler) SearchRecords(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := claimsFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := requirePatientAccess(claims, req.PatientId); err != nil {
		return nil, err
	}
	chunks, err := h.svc.SearchRecords(ctx, req.PatientId, req.Query, req.TopK)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	out := make([]*pb.RecordChunk, 0, len(chunks))
	for _, c := range chunks {
		out = append(out, &pb.RecordChunk{
			Text:       c.Text,
			SourceFile: c.SourceFile,
			Score:      c.Score,
		})
	}
	return &pb.SearchResponse{Chunks: out}, nil
}

func (h *GRPCHandler) HospitalSearchRecords(ctx context.Context, req *pb.HospitalSearchRequest) (*pb.SearchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := claimsFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.requireStaffPatientAccess(ctx, claims, req.PatientId, req.HospitalId); err != nil {
		return nil, err
	}
	chunks, err := h.svc.HospitalSearchRecords(ctx, req.PatientId, req.HospitalId, req.Query, req.TopK)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	out := make([]*pb.RecordChunk, 0, len(chunks))
	for _, c := range chunks {
		out = append(out, &pb.RecordChunk{
			Text:       c.Text,
			SourceFile: c.SourceFile,
			Score:      c.Score,
		})
	}
	return &pb.SearchResponse{Chunks: out}, nil
}

func (h *GRPCHandler) SummarizeRecords(ctx context.Context, req *pb.SummarizeRequest) (*pb.SummarizeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := claimsFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.requireStaffPatientAccess(ctx, claims, req.PatientId, req.HospitalId); err != nil {
		return nil, err
	}
	summary, err := h.svc.SummarizeRecords(ctx, req.PatientId, req.Focus)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.SummarizeResponse{Summary: summary}, nil
}

func (h *GRPCHandler) SaveConversationTurn(ctx context.Context, req *pb.SaveTurnRequest) (*pb.SaveTurnResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := claimsFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := requirePatientAccess(claims, req.PatientId); err != nil {
		return nil, err
	}
	if err := requireSessionMatch(claims, req.SessionId); err != nil {
		return nil, err
	}
	if err := h.svc.SaveConversationTurn(ctx, req.PatientId, req.SessionId, req.Role, req.Content); err != nil {
		return &pb.SaveTurnResponse{Ok: false}, status.Error(codes.Internal, err.Error())
	}
	return &pb.SaveTurnResponse{Ok: true}, nil
}

func (h *GRPCHandler) LoadRecentContext(ctx context.Context, req *pb.LoadContextRequest) (*pb.LoadContextResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := claimsFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := requirePatientAccess(claims, req.PatientId); err != nil {
		return nil, err
	}
	text, err := h.svc.LoadRecentContext(ctx, req.PatientId, req.Limit)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.LoadContextResponse{ContextText: text}, nil
}

func (h *GRPCHandler) GetPatientResources(ctx context.Context, req *pb.ResourceRequest) (*pb.ResourceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := claimsFromContext(ctx)
	if err != nil {
		return nil, err
	}

	switch claims.ActorType {
	case sharedauth.ActorPatient:
		if err := requirePatientAccess(claims, req.PatientId); err != nil {
			return nil, err
		}
	case sharedauth.ActorStaff:
		// Staff access to a target patient is allowed only after consent is verified
		// for claims.HospitalID and the requested patient ID.
		ok, err := sharedauth.CheckConsent(ctx, h.db, req.PatientId, claims.HospitalID)
		if err != nil {
			return nil, status.Error(codes.Internal, "consent check failed")
		}
		if !ok {
			return nil, status.Error(codes.PermissionDenied,
				"access denied: patient has not consented to share data with your hospital",
			)
		}
	default:
		return nil, status.Error(codes.PermissionDenied, "forbidden: unsupported actor type")
	}

	// Proto doesn't include pagination; pick a safe default.
	raws, err := h.svc.GetPatientResources(ctx, req.PatientId, req.ResourceType, req.StatusFilter, 100, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	out := make([]string, 0, len(raws))
	for _, r := range raws {
		out = append(out, string(r))
	}
	return &pb.ResourceResponse{
		ResourcesJson: out,
		ResourceType:  req.ResourceType,
		Total:         int32(len(out)),
	}, nil
}

func (h *GRPCHandler) IngestDocument(ctx context.Context, req *pb.IngestRequest) (*pb.IngestResponse, error) {
	// This core service currently supports IngestText (already-extracted text).
	// Text extraction (PDF/image) is intentionally left for a dedicated extractor.
	return nil, status.Error(codes.Unimplemented, "IngestDocument not implemented (text extraction not wired)")
}

func (h *GRPCHandler) IngestFHIRBundle(ctx context.Context, req *pb.FHIRBundleRequest) (*pb.IngestResponse, error) {
	// FHIR bundle parsing + dual-write (fhir_resources + record_chunks) can be added next.
	return nil, status.Error(codes.Unimplemented, "IngestFHIRBundle not implemented")
}

func claimsFromContext(ctx context.Context) (*sharedauth.Claims, error) {
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	return claims, nil
}

func requirePatientAccess(claims *sharedauth.Claims, patientID string) error {
	if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	if patientID != claims.PatientID {
		return status.Errorf(codes.PermissionDenied,
			"patient_id mismatch: token=%q request=%q",
			claims.PatientID, patientID,
		)
	}
	return nil
}

func requireSessionMatch(claims *sharedauth.Claims, sessionID string) error {
	if claims.SessionID != "" && sessionID != claims.SessionID {
		return status.Errorf(codes.PermissionDenied,
			"session_id mismatch: token=%q request=%q",
			claims.SessionID, sessionID,
		)
	}
	return nil
}

func (h *GRPCHandler) requireStaffPatientAccess(
	ctx context.Context,
	claims *sharedauth.Claims,
	patientID string,
	hospitalID string,
) error {
	if err := sharedauth.RequireActorType(claims, sharedauth.ActorStaff); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	if hospitalID != claims.HospitalID {
		return status.Errorf(codes.PermissionDenied,
			"hospital_id mismatch: token=%q request=%q",
			claims.HospitalID, hospitalID,
		)
	}
	ok, err := sharedauth.CheckConsent(ctx, h.db, patientID, claims.HospitalID)
	if err != nil {
		return status.Error(codes.Internal, "consent check failed")
	}
	if !ok {
		return status.Error(codes.PermissionDenied,
			"access denied: patient has not consented to share data with your hospital",
		)
	}
	return nil
}
