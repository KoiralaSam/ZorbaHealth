package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/ports/inbound"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_provider"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HealthProviderGRPCHandler struct {
	pb.UnimplementedHealthProviderServiceServer
	svc inbound.ProviderService
}

func NewHealthProviderGRPCHandler(server *grpc.Server, svc inbound.ProviderService) *HealthProviderGRPCHandler {
	h := &HealthProviderGRPCHandler{svc: svc}
	pb.RegisterHealthProviderServiceServer(server, h)
	return h
}

func (h *HealthProviderGRPCHandler) RegisterHospital(ctx context.Context, req *pb.RegisterHospitalRequest) (*pb.RegisterHospitalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	account, err := h.svc.RegisterHospital(ctx, models.HospitalRegistration{
		HospitalName: req.GetHospitalName(),
		LicenseNo:    req.GetLicenseNo(),
		StaffName:    req.GetStaffName(),
		StaffEmail:   req.GetEmail(),
		StaffPhone:   req.GetPhoneNumber(),
		StaffRole:    req.GetStaffRole(),
		Password:     req.GetPassword(),
	})
	if err != nil {
		return nil, mapRegisterError(err)
	}
	return &pb.RegisterHospitalResponse{
		Message:    "hospital staff registered successfully",
		UserId:     account.UserID,
		HospitalId: account.HospitalID,
		StaffId:    account.StaffID,
		StaffRole:  account.StaffRole,
	}, nil
}

func (h *HealthProviderGRPCHandler) RegisterHospitalStaff(ctx context.Context, req *pb.RegisterHospitalStaffRequest) (*pb.RegisterHospitalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	account, err := h.svc.RegisterHospitalStaff(ctx, models.HospitalStaffRegistration{
		HospitalID: req.GetHospitalId(),
		StaffName:  req.GetStaffName(),
		StaffEmail: req.GetEmail(),
		StaffPhone: req.GetPhoneNumber(),
		StaffRole:  req.GetStaffRole(),
		Password:   req.GetPassword(),
	})
	if err != nil {
		return nil, mapRegisterError(err)
	}
	return &pb.RegisterHospitalResponse{
		Message:    "hospital staff registered successfully",
		UserId:     account.UserID,
		HospitalId: account.HospitalID,
		StaffId:    account.StaffID,
		StaffRole:  account.StaffRole,
	}, nil
}

func mapRegisterError(err error) error {
	if isUniqueViolation(err) {
		return status.Error(codes.AlreadyExists, "email, phone number, or license number already registered")
	}
	msg := err.Error()
	if strings.Contains(msg, "required") || strings.Contains(msg, "inactive") || strings.Contains(msg, "invalid") {
		return status.Error(codes.InvalidArgument, msg)
	}
	// gRPC errors from auth passthrough
	if st, ok := status.FromError(err); ok {
		return st.Err()
	}
	return status.Error(codes.Internal, msg)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key")
}
