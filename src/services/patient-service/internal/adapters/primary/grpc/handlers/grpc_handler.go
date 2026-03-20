package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/inbound"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient"
	"github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedLoginServiceServer
	registration_verification.UnimplementedRegistrationVerificationServiceServer
	svc inbound.PatientService
}

func NewGRPCHandler(server *grpc.Server, svc inbound.PatientService) *gRPCHandler {
	handler := &gRPCHandler{
		svc: svc,
	}
	pb.RegisterLoginServiceServer(server, handler)
	registration_verification.RegisterRegistrationVerificationServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) StartRegistration(ctx context.Context, req *registration_verification.StartRegistrationRequest) (*registration_verification.StartRegistrationResponse, error) {
	var dateOfBirth time.Time
	if req.DateOfBirth != nil {
		dateOfBirth = req.DateOfBirth.AsTime()
	}
	registerReq := &models.RegisterPatientRequest{
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
		Password:    req.Password,
		FullName:    req.FullName,
		DateOfBirth: dateOfBirth,
	}

	// Check if phone number is already in use
	_, err := h.svc.GetPatientByPhoneNumber(ctx, registerReq.PhoneNumber)
	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "phone number is already in use")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.Internal, "failed to check existing phone number: "+err.Error())
	}

	// Check if email is already in use
	if registerReq.Email != "" {
		_, err = h.svc.GetPatientByEmail(ctx, registerReq.Email)
		if err == nil {
			return nil, status.Error(codes.AlreadyExists, "email is already in use")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.Internal, "failed to check existing email: "+err.Error())
		}
	}

	token, otp, err := h.svc.StartRegistrationWithVerification(ctx, registerReq)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to start registration with verification: "+err.Error())
	}
	_ = token
	_ = otp
	return &registration_verification.StartRegistrationResponse{Message: "Verification email sent. Please check your inbox."}, nil
}

func (h *gRPCHandler) VerifyEmail(ctx context.Context, req *registration_verification.VerifyEmailRequest) (*registration_verification.VerifyEmailResponse, error) {
	patient, err := h.svc.VerifyEmailAndCreatePatient(ctx, req.Token)
	if err != nil {
		//must send a not registered event message to ensure user is not registered in the system
		return nil, status.Error(codes.Internal, "Failed to verify email and create patient: "+err.Error())
	}
	return &registration_verification.VerifyEmailResponse{Message: "Email verified successfully", PatientId: patient.ID.String(), UserId: patient.UserID.String()}, nil
}

func (h *gRPCHandler) VerifyPhoneOTP(ctx context.Context, req *registration_verification.VerifyPhoneOTPRequest) (*registration_verification.VerifyPhoneOTPResponse, error) {
	if req.PhoneNumber == "" || req.Otp == "" {
		return nil, status.Error(codes.InvalidArgument, "phone_number and otp are required")
	}
	if err := h.svc.VerifyPhoneOTP(ctx, req.PhoneNumber, req.Otp); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Failed to verify OTP: "+err.Error())
	}
	return &registration_verification.VerifyPhoneOTPResponse{Message: "Phone verified successfully"}, nil
}

func (h *gRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Login not implemented")
}
