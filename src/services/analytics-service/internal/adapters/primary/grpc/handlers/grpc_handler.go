package grpc

import (
	"context"
	"errors"

	grpcmappers "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/adapters/primary/grpc/mappers"
	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/ports/inbound"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/analytics"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AnalyticsGRPCHandler struct {
	pb.UnimplementedAnalyticsServiceServer
	Service inbound.AnalyticsService
	DB      *pgxpool.Pool
}

func NewAnalyticsGRPCHandler(server *grpc.Server, svc inbound.AnalyticsService, db *pgxpool.Pool) *AnalyticsGRPCHandler {
	h := &AnalyticsGRPCHandler{Service: svc, DB: db}
	pb.RegisterAnalyticsServiceServer(server, h)
	return h
}

func (h *AnalyticsGRPCHandler) GetHospitalSummary(ctx context.Context, req *pb.HospitalSummaryRequest) (*pb.HospitalSummaryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if err := authorizeHospitalRequest(claims, req.HospitalId); err != nil {
		return nil, err
	}

	summary, err := h.Service.GetHospitalSummary(ctx, req.HospitalId)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return grpcmappers.SummaryToProto(summary), nil
}

func (h *AnalyticsGRPCHandler) GetCallVolume(ctx context.Context, req *pb.CallVolumeRequest) (*pb.CallVolumeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if err := authorizeHospitalRequest(claims, req.HospitalId); err != nil {
		return nil, err
	}

	points, err := h.Service.GetCallVolume(ctx, req.HospitalId, req.Period, req.Granularity)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return grpcmappers.CallVolumeToProto(points), nil
}

func (h *AnalyticsGRPCHandler) GetTopConditions(ctx context.Context, req *pb.TopConditionsRequest) (*pb.TopConditionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if err := authorizeHospitalRequest(claims, req.HospitalId); err != nil {
		return nil, err
	}

	conditions, err := h.Service.GetTopConditions(ctx, req.HospitalId, int(req.Limit))
	if err != nil {
		return nil, mapDomainError(err)
	}
	return grpcmappers.ConditionsToProto(conditions), nil
}

func (h *AnalyticsGRPCHandler) GetToolUsage(ctx context.Context, req *pb.ToolUsageRequest) (*pb.ToolUsageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if err := authorizeHospitalRequest(claims, req.HospitalId); err != nil {
		return nil, err
	}

	stats, err := h.Service.GetToolUsage(ctx, req.HospitalId, req.Period)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return grpcmappers.ToolUsageToProto(stats), nil
}

func (h *AnalyticsGRPCHandler) GetRecentActivity(ctx context.Context, req *pb.RecentActivityRequest) (*pb.RecentActivityResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if err := authorizeHospitalRequest(claims, req.HospitalId); err != nil {
		return nil, err
	}

	events, err := h.Service.GetRecentActivity(ctx, req.HospitalId, int(req.Limit))
	if err != nil {
		return nil, mapDomainError(err)
	}
	return grpcmappers.ActivityToProto(events), nil
}

func (h *AnalyticsGRPCHandler) GetPlatformSummary(ctx context.Context, req *pb.PlatformSummaryRequest) (*pb.PlatformSummaryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if err := requireAdmin(claims); err != nil {
		return nil, err
	}

	summary, err := h.Service.GetPlatformSummary(ctx)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return grpcmappers.PlatformSummaryToProto(summary), nil
}

func (h *AnalyticsGRPCHandler) GetHospitalBreakdown(ctx context.Context, req *pb.HospitalBreakdownRequest) (*pb.HospitalBreakdownResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if err := requireAdmin(claims); err != nil {
		return nil, err
	}

	stats, err := h.Service.GetHospitalBreakdown(ctx, int(req.Limit), req.Sort)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return grpcmappers.HospitalBreakdownToProto(stats), nil
}

func (h *AnalyticsGRPCHandler) GetPatientCallHistory(ctx context.Context, req *pb.PatientCallHistoryRequest) (*pb.PatientCallHistoryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if err := h.authorizePatientHistory(ctx, claims, req.PatientId); err != nil {
		return nil, err
	}

	history, err := h.Service.GetPatientCallHistory(ctx, req.PatientId, int(req.Limit))
	if err != nil {
		return nil, mapDomainError(err)
	}
	return grpcmappers.PatientCallHistoryToProto(history), nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domainerrors.ErrHospitalNotFound), errors.Is(err, domainerrors.ErrPatientNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domainerrors.ErrInvalidPeriod), errors.Is(err, domainerrors.ErrInvalidGranularity):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domainerrors.ErrUnauthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func authorizeHospitalRequest(claims *sharedauth.Claims, hospitalID string) error {
	switch claims.ActorType {
	case sharedauth.ActorStaff:
		if hospitalID != claims.HospitalID {
			return status.Errorf(codes.PermissionDenied, "hospital_id mismatch: token=%q request=%q", claims.HospitalID, hospitalID)
		}
		return nil
	case sharedauth.ActorAdmin:
		return nil
	case sharedauth.ActorPatient:
		return status.Error(codes.PermissionDenied, "patients cannot access hospital analytics")
	default:
		return status.Error(codes.PermissionDenied, "unsupported actor type")
	}
}

func requireAdmin(claims *sharedauth.Claims) error {
	if claims.ActorType != sharedauth.ActorAdmin {
		return status.Error(codes.PermissionDenied, "admin access required")
	}
	return nil
}

func (h *AnalyticsGRPCHandler) authorizePatientHistory(ctx context.Context, claims *sharedauth.Claims, patientID string) error {
	switch claims.ActorType {
	case sharedauth.ActorPatient:
		if patientID != claims.PatientID {
			return status.Errorf(codes.PermissionDenied, "patient_id mismatch: token=%q request=%q", claims.PatientID, patientID)
		}
		return nil
	case sharedauth.ActorStaff:
		allowed, err := sharedauth.CheckConsent(ctx, h.DB, patientID, claims.HospitalID)
		if err != nil {
			return status.Error(codes.Internal, "consent check failed")
		}
		if !allowed {
			const msg = "access denied: patient has not consented to share data with your hospital"
			sharedauth.LogAuditEventAsync(h.DB, claims, "GetPatientCallHistory", "consent-denied", msg)
			return status.Error(codes.PermissionDenied, msg)
		}
		// Staff access is allowed only after consent is verified against claims.HospitalID.
		return nil
	case sharedauth.ActorAdmin:
		return nil
	default:
		return status.Error(codes.PermissionDenied, "unsupported actor type")
	}
}
