package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/inbound"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient"
	"github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	patientportalpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patientportal"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type gRPCHandler struct {
	pb.UnimplementedLoginServiceServer
	patientportalpb.UnimplementedPatientPortalServiceServer
	registration_verification.UnimplementedRegistrationVerificationServiceServer
	svc inbound.PatientService
}

func NewGRPCHandler(server *grpc.Server, svc inbound.PatientService) *gRPCHandler {
	handler := &gRPCHandler{
		svc: svc,
	}
	pb.RegisterLoginServiceServer(server, handler)
	patientportalpb.RegisterPatientPortalServiceServer(server, handler)
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
		Message: "Verification email sent. Please check your inbox and verify your phone with the SMS code.",
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
	session, err := h.svc.VerifyExistingPhoneOTP(ctx, req.PhoneNumber, req.Otp)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Failed to verify OTP: "+err.Error())
	}
	return &registration_verification.VerifyExistingPhoneOTPResponse{
		Message:     "Existing patient verified successfully",
		PatientId:   session.PatientID,
		AccessToken: session.AccessToken,
	}, nil
}

func (h *gRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	session, err := h.svc.LoginPatient(ctx, &models.Patient{
		PhoneNumber:  req.PhoneNumber,
		Email:        req.Email,
		MedicalNotes: req.Password,
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "patient login failed: "+err.Error())
	}
	return &pb.LoginResponse{
		Message:     "patient login successful",
		AccessToken: session.AccessToken,
		PatientID:   session.PatientID,
	}, nil
}

func (h *gRPCHandler) GetProfile(ctx context.Context, req *patientportalpb.GetPatientProfileRequest) (*patientportalpb.GetPatientProfileResponse, error) {
	if req == nil || req.GetPatientId() == "" {
		return nil, status.Error(codes.InvalidArgument, "patient_id is required")
	}
	profile, err := h.svc.GetPatientProfile(ctx, req.GetPatientId())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "patient not found")
		}
		return nil, status.Error(codes.Internal, "failed to load patient profile: "+err.Error())
	}
	return &patientportalpb.GetPatientProfileResponse{
		Profile: &patientportalpb.PatientProfile{
			PatientId:    profile.PatientID,
			PhoneNumber:  profile.PhoneNumber,
			Email:        profile.Email,
			FullName:     profile.FullName,
			DateOfBirth:  dateOnly(profile.DateOfBirth),
			MedicalNotes: profile.MedicalNotes,
		},
	}, nil
}

func (h *gRPCHandler) ListCallSummaries(ctx context.Context, req *patientportalpb.ListPatientCallSummariesRequest) (*patientportalpb.ListPatientCallSummariesResponse, error) {
	if req == nil || req.GetPatientId() == "" {
		return nil, status.Error(codes.InvalidArgument, "patient_id is required")
	}
	calls, err := h.svc.ListPatientCallSummaries(ctx, req.GetPatientId(), req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load patient calls: "+err.Error())
	}
	out := make([]*patientportalpb.PatientCallSummary, 0, len(calls))
	for _, call := range calls {
		out = append(out, &patientportalpb.PatientCallSummary{
			Id:            call.ID,
			Status:        call.Status,
			StartedAt:     timeToProto(call.StartedAt),
			EndedAt:       timeToProto(call.EndedAt),
			RecordingUrl:  call.RecordingURL,
			Summary:       call.Summary,
			LivekitRoomId: call.LivekitRoomID,
		})
	}
	return &patientportalpb.ListPatientCallSummariesResponse{Calls: out}, nil
}

func (h *gRPCHandler) CreateWelfareCheck(ctx context.Context, req *patientportalpb.CreateWelfareCheckRequest) (*patientportalpb.CreateWelfareCheckResponse, error) {
	if req == nil || strings.TrimSpace(req.GetPatientId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "patient_id is required")
	}
	if req.GetScheduledAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "scheduled_at is required")
	}
	patientID, err := uuid.Parse(strings.TrimSpace(req.GetPatientId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid patient_id")
	}
	check, err := h.svc.CreateWelfareCheck(ctx, &models.CreateWelfareCheckCommand{
		PatientID:    patientID,
		ScheduledAt:  req.GetScheduledAt().AsTime(),
		Timezone:     strings.TrimSpace(req.GetTimezone()),
		ReasonCode:   models.WelfareCheckReason(strings.TrimSpace(req.GetReasonCode())),
		ReasonDetail: req.GetReasonDetail(),
		ActorID:      patientID.String(),
	})
	if err != nil {
		return nil, mapWelfareError(err)
	}
	return &patientportalpb.CreateWelfareCheckResponse{WelfareCheck: welfareCheckToProto(check)}, nil
}

func (h *gRPCHandler) ListWelfareChecks(ctx context.Context, req *patientportalpb.ListWelfareChecksRequest) (*patientportalpb.ListWelfareChecksResponse, error) {
	if req == nil || strings.TrimSpace(req.GetPatientId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "patient_id is required")
	}
	patientID, err := uuid.Parse(strings.TrimSpace(req.GetPatientId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid patient_id")
	}
	checks, err := h.svc.ListWelfareChecks(ctx, models.ListWelfareChecksFilter{
		PatientID:        patientID,
		IncludeCancelled: req.GetIncludeCancelled(),
		Limit:            req.GetLimit(),
	})
	if err != nil {
		return nil, mapWelfareError(err)
	}
	out := make([]*patientportalpb.WelfareCheck, 0, len(checks))
	for i := range checks {
		out = append(out, welfareCheckToProto(&checks[i]))
	}
	return &patientportalpb.ListWelfareChecksResponse{WelfareChecks: out}, nil
}

func (h *gRPCHandler) CancelWelfareCheck(ctx context.Context, req *patientportalpb.CancelWelfareCheckRequest) (*patientportalpb.CancelWelfareCheckResponse, error) {
	if req == nil || strings.TrimSpace(req.GetPatientId()) == "" || strings.TrimSpace(req.GetWelfareCheckId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "patient_id and welfare_check_id are required")
	}
	check, err := h.svc.CancelWelfareCheck(ctx, req.GetPatientId(), req.GetWelfareCheckId())
	if err != nil {
		return nil, mapWelfareError(err)
	}
	return &patientportalpb.CancelWelfareCheckResponse{WelfareCheck: welfareCheckToProto(check)}, nil
}

func (h *gRPCHandler) UpdateWelfareRunLifecycle(ctx context.Context, req *patientportalpb.UpdateWelfareRunLifecycleRequest) (*patientportalpb.UpdateWelfareRunLifecycleResponse, error) {
	if req == nil || strings.TrimSpace(req.GetPatientId()) == "" || strings.TrimSpace(req.GetRunId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "patient_id and run_id are required")
	}
	run, err := h.svc.UpdateWelfareRunLifecycle(ctx, &models.UpdateWelfareRunLifecycleCommand{
		PatientID: strings.TrimSpace(req.GetPatientId()),
		RunID:     strings.TrimSpace(req.GetRunId()),
		Status:    models.WelfareCheckRunStatus(strings.TrimSpace(req.GetStatus())),
		Reason:    strings.TrimSpace(req.GetReason()),
	})
	if err != nil {
		return nil, mapWelfareError(err)
	}
	return &patientportalpb.UpdateWelfareRunLifecycleResponse{Run: welfareCheckRunToProto(run)}, nil
}

func welfareCheckToProto(check *models.WelfareCheck) *patientportalpb.WelfareCheck {
	if check == nil {
		return nil
	}
	return &patientportalpb.WelfareCheck{
		Id:                     check.ID.String(),
		PatientId:              check.PatientID.String(),
		ScheduledAt:            timestamppb.New(check.ScheduledAt),
		Timezone:               check.Timezone,
		ReasonCode:             string(check.ReasonCode),
		ReasonDetail:           check.ReasonDetail,
		Status:                 string(check.Status),
		RecurrenceRule:         check.RecurrenceRule,
		CreatedAt:              timestamppb.New(check.CreatedAt),
		UpdatedAt:              timestamppb.New(check.UpdatedAt),
		CancelledAt:            timePtrToProto(check.CancelledAt),
		LatestRunId:            check.LatestRunID,
		LatestRunStatus:        check.LatestRunStatus,
		LatestRunAttempts:      check.LatestRunAttempts,
		LatestRunFailureReason: check.LatestRunFailureReason,
	}
}

func welfareCheckRunToProto(run *models.WelfareCheckRun) *patientportalpb.WelfareCheckRun {
	if run == nil {
		return nil
	}
	return &patientportalpb.WelfareCheckRun{
		Id:              run.ID.String(),
		RequestId:       run.RequestID.String(),
		PatientId:       run.PatientID.String(),
		Status:          string(run.Status),
		Attempts:        run.Attempts,
		FailureReason:   run.FailureReason,
		LivekitRoomName: run.LiveKitRoomName,
		ScheduledAt:     timestamppb.New(run.ScheduledAt),
		UpdatedAt:       timestamppb.New(run.UpdatedAt),
	}
}

func mapWelfareError(err error) error {
	switch {
	case errors.Is(err, domainErrors.ErrWelfareCheckNotFound),
		errors.Is(err, domainErrors.ErrWelfareCheckRunNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domainErrors.ErrWelfareCheckStartsAtInvalid),
		errors.Is(err, domainErrors.ErrWelfareCheckTimezoneInvalid),
		errors.Is(err, domainErrors.ErrWelfareCheckReasonInvalid),
		errors.Is(err, domainErrors.ErrWelfareCheckReasonRequired),
		errors.Is(err, domainErrors.ErrWelfareCheckReasonTooLong),
		errors.Is(err, domainErrors.ErrWelfareCheckPhoneRequired),
		errors.Is(err, domainErrors.ErrWelfareCheckRunTransition):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domainErrors.ErrWelfareCheckConsentRequired):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domainErrors.ErrWelfareCheckConsentUnavailable),
		errors.Is(err, domainErrors.ErrWelfareCheckDispatchUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func timeToProto(value *time.Time) *timestamppb.Timestamp {
	if value == nil || value.IsZero() {
		return nil
	}
	return timestamppb.New(*value)
}

func timePtrToProto(value *time.Time) *timestamppb.Timestamp {
	return timeToProto(value)
}

func dateOnly(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}
