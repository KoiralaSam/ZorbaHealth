package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/ports/inbound"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/auth"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthGRPCHandler implements auth.proto LoginService, RegisterPatientService, RegisterHealthProviderService, VerifyTokenService, LogoutService.
type AuthGRPCHandler struct {
	pb.UnimplementedLoginServiceServer
	pb.UnimplementedRegisterPatientServiceServer
	pb.UnimplementedRegisterHealthProviderServiceServer
	pb.UnimplementedVerifyTokenServiceServer
	pb.UnimplementedLogoutServiceServer
	svc inbound.AuthService
}

// NewAuthGRPCHandler registers all auth gRPC services on the given server.
func NewAuthGRPCHandler(server *grpc.Server, svc inbound.AuthService) *AuthGRPCHandler {
	h := &AuthGRPCHandler{svc: svc}
	pb.RegisterLoginServiceServer(server, h)
	pb.RegisterRegisterPatientServiceServer(server, h)
	pb.RegisterRegisterHealthProviderServiceServer(server, h)
	pb.RegisterVerifyTokenServiceServer(server, h)
	pb.RegisterLogoutServiceServer(server, h)
	return h
}

// Login implements auth.LoginService.
func (h *AuthGRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	token, userID, role, err := h.svc.Login(ctx, req.Email, req.PhoneNumber, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &pb.LoginResponse{
		Message:     "login successful",
		AccessToken: token,
		UserId:      userID,
		Role:        role,
	}, nil
}

// RegisterPatient implements auth.RegisterPatientService. Creates user with role=patient only; patient-service creates the patient row.
func (h *AuthGRPCHandler) RegisterPatient(ctx context.Context, req *pb.RegisterPatientRequest) (*pb.RegisterPatientResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	user, err := h.svc.RegisterUser(ctx, req.Email, req.PhoneNumber, req.Password, "patient")
	if err != nil {
		if isUniqueViolation(err) {
			h.svc.DeleteUser(ctx, user.ID)
			return nil, status.Error(codes.AlreadyExists, "email or phone number already registered")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.RegisterPatientResponse{
		Message: "user registered successfully",
		UserId:  user.ID,
	}, nil
}

// RegisterHealthProvider implements auth.RegisterHealthProviderService. Creates user with role=health_provider only.
func (h *AuthGRPCHandler) RegisterHealthProvider(ctx context.Context, req *pb.RegisterHealthProviderRequest) (*pb.RegisterHealthProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	user, err := h.svc.RegisterUser(ctx, req.Email, req.PhoneNumber, req.Password, "health_provider")
	if err != nil {
		if isUniqueViolation(err) {
			h.svc.DeleteUser(ctx, user.ID)
			return nil, status.Error(codes.AlreadyExists, "email or phone number already registered")
		}

		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.RegisterHealthProviderResponse{
		Message: "user registered successfully",
		UserId:  user.ID,
	}, nil
}

// VerifyToken implements auth.VerifyTokenService. Middleware as a service: other services call this to validate JWT and get claims.
func (h *AuthGRPCHandler) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	if req == nil || req.AccessToken == "" {
		return &pb.VerifyTokenResponse{Valid: false, Message: "token required"}, nil
	}
	userID, authUUID, role, err := h.svc.VerifyToken(ctx, req.AccessToken)
	if err != nil {
		return &pb.VerifyTokenResponse{Valid: false, Message: err.Error()}, nil
	}
	return &pb.VerifyTokenResponse{
		Valid:    true,
		UserId:   userID,
		AuthUuid: authUUID,
		Role:     role,
	}, nil
}

// Logout implements auth.LogoutService. Invalidates the session (deletes auth row).
func (h *AuthGRPCHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if req == nil || req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "access_token required")
	}
	msg, err := h.svc.Logout(ctx, req.AccessToken)
	if err != nil {
		return &pb.LogoutResponse{Message: msg}, status.Error(codes.Unauthenticated, err.Error())
	}
	return &pb.LogoutResponse{Message: msg}, nil
}

// isUniqueViolation returns true if err is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	// Fallback: check error message in case error is wrapped without As support
	return strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key")
}
