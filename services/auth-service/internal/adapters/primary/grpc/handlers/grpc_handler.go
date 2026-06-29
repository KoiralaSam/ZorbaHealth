package grpc

import (
	"context"
	"errors"
	"strings"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/domain/models"
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
	pb.UnimplementedCreatePatientSessionServiceServer
	pb.UnimplementedVerifyTokenServiceServer
	pb.UnimplementedLogoutServiceServer
	pb.UnimplementedRefreshSessionServiceServer
	pb.UnimplementedUserCredentialsServiceServer
	svc inbound.AuthService
}

// NewAuthGRPCHandler registers all auth gRPC services on the given server.
func NewAuthGRPCHandler(server *grpc.Server, svc inbound.AuthService) *AuthGRPCHandler {
	h := &AuthGRPCHandler{svc: svc}
	pb.RegisterLoginServiceServer(server, h)
	pb.RegisterRegisterPatientServiceServer(server, h)
	pb.RegisterRegisterHealthProviderServiceServer(server, h)
	pb.RegisterCreatePatientSessionServiceServer(server, h)
	pb.RegisterVerifyTokenServiceServer(server, h)
	pb.RegisterLogoutServiceServer(server, h)
	pb.RegisterRefreshSessionServiceServer(server, h)
	pb.RegisterUserCredentialsServiceServer(server, h)
	return h
}

func (h *AuthGRPCHandler) CreatePatientSession(ctx context.Context, req *pb.CreatePatientSessionRequest) (*pb.CreatePatientSessionResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id required")
	}
	access, refresh, auth, err := h.svc.CreatePatientSession(ctx, req.UserId, req.Scopes)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreatePatientSessionResponse{
		Message:      "patient session created",
		AccessToken:  access,
		UserId:       auth.UserID,
		AuthUuid:     auth.AuthUUID,
		RefreshToken: refresh,
	}, nil
}

// Login implements auth.LoginService.
func (h *AuthGRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	access, refresh, userID, role, authUUID, err := h.svc.Login(ctx, req.Email, req.PhoneNumber, req.Password, "")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &pb.LoginResponse{
		Message:      "login successful",
		AccessToken:  access,
		UserId:       userID,
		Role:         role,
		RefreshToken: refresh,
		AuthUuid:     authUUID,
	}, nil
}

func (h *AuthGRPCHandler) ValidateUserCredentials(ctx context.Context, req *pb.LoginRequest) (*pb.ValidateUserCredentialsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	userID, role, err := h.svc.ValidateUserCredentials(ctx, req.Email, req.PhoneNumber, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &pb.ValidateUserCredentialsResponse{UserId: userID, Role: role}, nil
}

func (h *AuthGRPCHandler) RefreshSession(ctx context.Context, req *pb.RefreshSessionRequest) (*pb.RefreshSessionResponse, error) {
	if req == nil || strings.TrimSpace(req.RefreshToken) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token required")
	}
	access, refresh, userID, authUUID, role, err := h.svc.RefreshSession(ctx, req.RefreshToken, req.ExpectedActorType, req.ClientKind)
	if err != nil {
		if errors.Is(err, domainerrors.ErrRefreshTokenReuse) {
			return nil, status.Error(codes.Unauthenticated, "REFRESH_TOKEN_REUSE")
		}
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &pb.RefreshSessionResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		UserId:       userID,
		AuthUuid:     authUUID,
		Role:         role,
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

// RegisterHealthProvider implements auth.RegisterHealthProviderService. It creates
// a hospital plus initial staff when hospital_id is empty, or additional staff for
// an existing hospital when hospital_id is provided.
func (h *AuthGRPCHandler) RegisterHealthProvider(ctx context.Context, req *pb.RegisterHealthProviderRequest) (*pb.RegisterHealthProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	var (
		account *models.HospitalStaffAccount
		err     error
	)
	if strings.TrimSpace(req.GetHospitalId()) != "" {
		account, err = h.svc.RegisterHospitalStaff(
			ctx,
			req.GetHospitalId(),
			req.GetStaffName(),
			req.GetEmail(),
			req.GetPhoneNumber(),
			req.GetPassword(),
			req.GetStaffRole(),
		)
	} else {
		account, err = h.svc.RegisterHospital(
			ctx,
			req.GetOrganizationName(),
			req.GetLicenseNo(),
			req.GetStaffName(),
			req.GetEmail(),
			req.GetPhoneNumber(),
			req.GetPassword(),
			req.GetStaffRole(),
		)
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, status.Error(codes.AlreadyExists, "email, phone number, or license number already registered")
		}

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pb.RegisterHealthProviderResponse{
		Message:    "hospital staff registered successfully",
		UserId:     account.UserID,
		HospitalId: account.HospitalID,
		StaffId:    account.StaffID,
		StaffRole:  account.StaffRole,
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
