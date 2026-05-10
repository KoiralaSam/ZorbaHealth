package grpc

import (
	"context"
	"errors"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
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
	if !errors.Is(err, pgx.ErrNoRows) && !errors.Is(err, domainErrors.ErrAmbiguousPhoneNumber) {
		return nil, status.Error(codes.Internal, "failed to check existing phone number: "+err.Error())
	}
	if errors.Is(err, domainErrors.ErrAmbiguousPhoneNumber) {
		return nil, status.Error(codes.AlreadyExists, "phone number is already in use")
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
	return &registration_verification.StartRegistrationResponse{
		Message:           "Verification started. Ask the caller for the SMS code to continue.",
		RegistrationToken: token,
	}, nil
}

func (h *gRPCHandler) LookupPatientByPhone(ctx context.Context, req *registration_verification.LookupPatientByPhoneRequest) (*registration_verification.LookupPatientByPhoneResponse, error) {
	if req.PhoneNumber == "" {
		return nil, status.Error(codes.InvalidArgument, "phone_number is required")
	}
	patient, err := h.svc.GetPatientByPhoneNumber(ctx, req.PhoneNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &registration_verification.LookupPatientByPhoneResponse{Found: false}, nil
		}
		if errors.Is(err, domainErrors.ErrAmbiguousPhoneNumber) {
			return nil, status.Error(codes.FailedPrecondition, "multiple patients found for phone number")
		}
		return nil, status.Error(codes.Internal, "failed to lookup patient by phone: "+err.Error())
	}
	return &registration_verification.LookupPatientByPhoneResponse{
		Found:     true,
		PatientId: patient.ID.String(),
		FullName:  patient.FullName,
	}, nil
}

func (h *gRPCHandler) StartExistingPhoneVerification(ctx context.Context, req *registration_verification.StartExistingPhoneVerificationRequest) (*registration_verification.StartExistingPhoneVerificationResponse, error) {
	if req.PhoneNumber == "" {
		return nil, status.Error(codes.InvalidArgument, "phone_number is required")
	}
	if err := h.svc.StartExistingPhoneVerification(ctx, req.PhoneNumber); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, domainErrors.ErrExistingPatientNotFound) {
			return nil, status.Error(codes.NotFound, "patient not found for phone number")
		}
		if errors.Is(err, domainErrors.ErrAmbiguousPhoneNumber) {
			return nil, status.Error(codes.FailedPrecondition, "multiple patients found for phone number")
		}
		return nil, status.Error(codes.Internal, "failed to start phone verification: "+err.Error())
	}
	return &registration_verification.StartExistingPhoneVerificationResponse{
		Message: "Verification code sent",
	}, nil
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

func (h *gRPCHandler) VerifyExistingPhoneOTP(ctx context.Context, req *registration_verification.VerifyExistingPhoneOTPRequest) (*registration_verification.VerifyExistingPhoneOTPResponse, error) {
	if req.PhoneNumber == "" || req.Otp == "" {
		return nil, status.Error(codes.InvalidArgument, "phone_number and otp are required")
	}
	patient, err := h.svc.VerifyExistingPhoneOTP(ctx, req.PhoneNumber, req.Otp)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Failed to verify OTP: "+err.Error())
	}
	return &registration_verification.VerifyExistingPhoneOTPResponse{
		Message:   "Existing patient verified successfully",
		PatientId: patient.ID.String(),
	}, nil
}

func (h *gRPCHandler) CompletePhoneRegistration(ctx context.Context, req *registration_verification.CompletePhoneRegistrationRequest) (*registration_verification.CompletePhoneRegistrationResponse, error) {
	if req.RegistrationToken == "" {
		return nil, status.Error(codes.InvalidArgument, "registration_token is required")
	}
	patient, err := h.svc.CompletePhoneRegistration(ctx, req.RegistrationToken)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Failed to complete phone registration: "+err.Error())
	}
	return &registration_verification.CompletePhoneRegistrationResponse{
		Message:   "Phone registration completed successfully",
		PatientId: patient.ID.String(),
	}, nil
}

func (h *gRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Login not implemented")
}
