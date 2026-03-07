package grpc

import (
	"context"
	"time"

	events "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/primary/events/rabbitmq"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/services"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient"
	"github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedLoginServiceServer
	registration_verification.UnimplementedRegistrationVerificationServiceServer
	svc              *services.PatientService
	patientPublisher *events.PatientPublisher
}

func NewGRPCHandler(server *grpc.Server, svc *services.PatientService, patientPublisher *events.PatientPublisher) *gRPCHandler {
	handler := &gRPCHandler{
		svc:              svc,
		patientPublisher: patientPublisher,
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
	token, err := h.svc.StartRegistrationWithVerification(ctx, registerReq)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to start registration with verification: "+err.Error())
	}
	if err := h.patientPublisher.PublishPatientChached(ctx, registerReq, token); err != nil {
		return nil, status.Error(codes.Internal, "Failed to publish patient registered event: "+err.Error())
	}
	return &registration_verification.StartRegistrationResponse{Message: "Verification email sent. Please check your inbox."}, nil
}

func (h *gRPCHandler) VerifyEmail(ctx context.Context, req *registration_verification.VerifyEmailRequest) (*registration_verification.VerifyEmailResponse, error) {
	patient, err := h.svc.VerifyEmailAndCreatePatient(ctx, req.Token)
	if err != nil {
		//must send a not registered event message to ensure user is not registered in the system
		return nil, status.Error(codes.Internal, "Failed to verify email and create patient: "+err.Error())
	}
	if err := h.patientPublisher.PublishPatientRegistered(ctx, patient); err != nil {
		return nil, status.Error(codes.Internal, "Failed to publish patient registered event: "+err.Error())
	}
	return &registration_verification.VerifyEmailResponse{Message: "Email verified successfully", PatientId: patient.ID.String(), UserId: patient.UserID.String()}, nil
}

func (h *gRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method Login not implemented")
}
