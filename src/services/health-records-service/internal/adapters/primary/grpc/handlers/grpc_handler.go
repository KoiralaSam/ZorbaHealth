package handlers

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/inbound"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCHandler is the primary gRPC adapter for health-records-service.
// It maps protobuf requests/responses to the inbound core service.
type GRPCHandler struct {
	pb.UnimplementedHealthRecordServiceServer
	svc inbound.HealthRecordsService
}

func NewGRPCHandler(server *grpc.Server, svc inbound.HealthRecordsService) *GRPCHandler {
	h := &GRPCHandler{svc: svc}
	pb.RegisterHealthRecordServiceServer(server, h)
	return h
}

func (h *GRPCHandler) SearchRecords(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
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
	if err := h.svc.SaveConversationTurn(ctx, req.PatientId, req.SessionId, req.Role, req.Content); err != nil {
		return &pb.SaveTurnResponse{Ok: false}, status.Error(codes.Internal, err.Error())
	}
	return &pb.SaveTurnResponse{Ok: true}, nil
}

func (h *GRPCHandler) LoadRecentContext(ctx context.Context, req *pb.LoadContextRequest) (*pb.LoadContextResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
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
