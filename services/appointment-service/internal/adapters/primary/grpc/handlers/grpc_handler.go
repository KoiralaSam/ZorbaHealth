package handlers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/adapters/primary/grpc/mappers"
	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/inbound"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	appointmentpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/appointment"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppointmentGRPCHandler struct {
	appointmentpb.UnimplementedAppointmentServiceServer
	appointments inbound.AppointmentService
	availability inbound.AvailabilityService
}

func NewAppointmentGRPCHandler(server *grpc.Server, appointments inbound.AppointmentService, availability inbound.AvailabilityService) {
	h := &AppointmentGRPCHandler{appointments: appointments, availability: availability}
	appointmentpb.RegisterAppointmentServiceServer(server, h)
}

func (h *AppointmentGRPCHandler) SetAvailabilityRules(ctx context.Context, req *appointmentpb.SetAvailabilityRulesRequest) (*appointmentpb.SetAvailabilityRulesResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	staffID, err := uuid.Parse(req.GetStaffId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
	}
	hospitalID, err := uuid.Parse(req.GetHospitalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
	}
	rules := make([]models.AvailabilityRule, 0, len(req.GetRules()))
	for _, r := range req.GetRules() {
		rules = append(rules, mappers.RuleFromProto(r, staffID, hospitalID))
	}
	out, err := h.availability.SetAvailabilityRules(ctx, actor, staffID, hospitalID, rules)
	if err != nil {
		return nil, mapError(err)
	}
	protoRules := make([]*appointmentpb.AvailabilityRule, 0, len(out))
	for _, r := range out {
		protoRules = append(protoRules, mappers.RuleToProto(r))
	}
	return &appointmentpb.SetAvailabilityRulesResponse{Rules: protoRules}, nil
}

func (h *AppointmentGRPCHandler) GetAvailabilityRules(ctx context.Context, req *appointmentpb.GetAvailabilityRulesRequest) (*appointmentpb.GetAvailabilityRulesResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	staffID, err := uuid.Parse(req.GetStaffId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
	}
	hospitalID, err := uuid.Parse(req.GetHospitalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
	}
	out, err := h.availability.GetAvailabilityRules(ctx, actor, staffID, hospitalID)
	if err != nil {
		return nil, mapError(err)
	}
	protoRules := make([]*appointmentpb.AvailabilityRule, 0, len(out))
	for _, r := range out {
		protoRules = append(protoRules, mappers.RuleToProto(r))
	}
	return &appointmentpb.GetAvailabilityRulesResponse{Rules: protoRules}, nil
}

func (h *AppointmentGRPCHandler) AddAvailabilityException(ctx context.Context, req *appointmentpb.AddAvailabilityExceptionRequest) (*appointmentpb.AddAvailabilityExceptionResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	staffID, err := uuid.Parse(req.GetStaffId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
	}
	hospitalID, err := uuid.Parse(req.GetHospitalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
	}
	if req.GetStartsAt() == nil || req.GetEndsAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "starts_at and ends_at required")
	}
	ex := models.AvailabilityException{
		StaffID:     staffID,
		HospitalID:  hospitalID,
		StartsAt:    req.GetStartsAt().AsTime(),
		EndsAt:      req.GetEndsAt().AsTime(),
		Reason:      req.GetReason(),
		IsAvailable: req.GetIsAvailable(),
	}
	created, err := h.availability.AddAvailabilityException(ctx, actor, ex)
	if err != nil {
		return nil, mapError(err)
	}
	return &appointmentpb.AddAvailabilityExceptionResponse{Exception: mappers.ExceptionToProto(*created)}, nil
}

func (h *AppointmentGRPCHandler) RemoveAvailabilityException(ctx context.Context, req *appointmentpb.RemoveAvailabilityExceptionRequest) (*appointmentpb.RemoveAvailabilityExceptionResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetExceptionId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid exception_id")
	}
	if err := h.availability.RemoveAvailabilityException(ctx, actor, id); err != nil {
		return nil, mapError(err)
	}
	return &appointmentpb.RemoveAvailabilityExceptionResponse{Removed: true}, nil
}

func (h *AppointmentGRPCHandler) ListAvailabilityExceptions(ctx context.Context, req *appointmentpb.ListAvailabilityExceptionsRequest) (*appointmentpb.ListAvailabilityExceptionsResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	staffID, err := uuid.Parse(req.GetStaffId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
	}
	hospitalID, err := uuid.Parse(req.GetHospitalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
	}
	from := time.Now().UTC()
	to := from.AddDate(0, 1, 0)
	if req.GetFrom() != nil {
		from = req.GetFrom().AsTime()
	}
	if req.GetTo() != nil {
		to = req.GetTo().AsTime()
	}
	out, err := h.availability.ListAvailabilityExceptions(ctx, actor, staffID, hospitalID, from, to)
	if err != nil {
		return nil, mapError(err)
	}
	items := make([]*appointmentpb.AvailabilityException, 0, len(out))
	for _, e := range out {
		items = append(items, mappers.ExceptionToProto(e))
	}
	return &appointmentpb.ListAvailabilityExceptionsResponse{Exceptions: items}, nil
}

func (h *AppointmentGRPCHandler) ListAvailableSlots(ctx context.Context, req *appointmentpb.ListAvailableSlotsRequest) (*appointmentpb.ListAvailableSlotsResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	staffID, err := uuid.Parse(req.GetStaffId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
	}
	hospitalID, err := uuid.Parse(req.GetHospitalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
	}
	from := time.Now().UTC()
	to := from.AddDate(0, 0, 14)
	if req.GetFrom() != nil {
		from = req.GetFrom().AsTime()
	}
	if req.GetTo() != nil {
		to = req.GetTo().AsTime()
	}
	slots, err := h.appointments.ListAvailableSlots(ctx, actor, staffID, hospitalID, from, to, req.GetLimit())
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*appointmentpb.AppointmentSlot, 0, len(slots))
	for _, s := range slots {
		out = append(out, mappers.SlotToProto(s))
	}
	return &appointmentpb.ListAvailableSlotsResponse{Slots: out}, nil
}

func (h *AppointmentGRPCHandler) GetNextAvailableSlot(ctx context.Context, req *appointmentpb.GetNextAvailableSlotRequest) (*appointmentpb.GetNextAvailableSlotResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	staffID, err := uuid.Parse(req.GetStaffId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
	}
	hospitalID, err := uuid.Parse(req.GetHospitalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
	}
	after := time.Now().UTC()
	if req.GetAfter() != nil {
		after = req.GetAfter().AsTime()
	}
	slot, err := h.appointments.GetNextAvailableSlot(ctx, actor, staffID, hospitalID, after)
	if err != nil {
		return nil, mapError(err)
	}
	return &appointmentpb.GetNextAvailableSlotResponse{Slot: mappers.SlotToProto(*slot)}, nil
}

func (h *AppointmentGRPCHandler) BookAppointment(ctx context.Context, req *appointmentpb.BookAppointmentRequest) (*appointmentpb.BookAppointmentResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	patientID, err := uuid.Parse(req.GetPatientId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid patient_id")
	}
	staffID, err := uuid.Parse(req.GetStaffId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
	}
	hospitalID, err := uuid.Parse(req.GetHospitalId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
	}
	if req.GetStartsAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "starts_at required")
	}
	corrID := uuid.New()
	if strings.TrimSpace(req.GetCorrelationId()) != "" {
		if parsed, perr := uuid.Parse(req.GetCorrelationId()); perr == nil {
			corrID = parsed
		}
	}
	cmd := &models.BookAppointmentCommand{
		PatientID:         patientID,
		StaffID:           staffID,
		HospitalID:        hospitalID,
		StartsAt:          req.GetStartsAt().AsTime(),
		DurationMinutes:   req.GetDurationMinutes(),
		Timezone:          req.GetTimezone(),
		Type:              models.AppointmentType(req.GetType()),
		Channel:           models.AppointmentChannel(req.GetChannel()),
		Title:             req.GetTitle(),
		Notes:             req.GetNotes(),
		CorrelationID:     corrID,
		VoiceSessionID:    req.GetVoiceSessionId(),
		BookedByActorType: req.GetBookedByActorType(),
		BookedByActorID:   req.GetBookedByActorId(),
		SendSMS:           req.GetSendSms(),
		SendEmail:         req.GetSendEmail(),
	}
	appt, err := h.appointments.BookAppointment(ctx, actor, cmd)
	if err != nil {
		return nil, mapError(err)
	}
	return &appointmentpb.BookAppointmentResponse{Appointment: mappers.AppointmentToProto(appt)}, nil
}

func (h *AppointmentGRPCHandler) RescheduleAppointment(ctx context.Context, req *appointmentpb.RescheduleAppointmentRequest) (*appointmentpb.RescheduleAppointmentResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetAppointmentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid appointment_id")
	}
	if req.GetStartsAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "starts_at required")
	}
	appt, err := h.appointments.RescheduleAppointment(ctx, actor, id, req.GetStartsAt().AsTime(), req.GetDurationMinutes(), req.GetTimezone(), req.GetTitle())
	if err != nil {
		return nil, mapError(err)
	}
	return &appointmentpb.RescheduleAppointmentResponse{Appointment: mappers.AppointmentToProto(appt)}, nil
}

func (h *AppointmentGRPCHandler) CancelAppointment(ctx context.Context, req *appointmentpb.CancelAppointmentRequest) (*appointmentpb.CancelAppointmentResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetAppointmentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid appointment_id")
	}
	appt, err := h.appointments.CancelAppointment(ctx, actor, id, req.GetReason())
	if err != nil {
		return nil, mapError(err)
	}
	return &appointmentpb.CancelAppointmentResponse{Appointment: mappers.AppointmentToProto(appt)}, nil
}

func (h *AppointmentGRPCHandler) ListAppointments(ctx context.Context, req *appointmentpb.ListAppointmentsRequest) (*appointmentpb.ListAppointmentsResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	filter := models.ListAppointmentsFilter{
		IncludeCancelled: req.GetIncludeCancelled(),
		Limit:            req.GetLimit(),
	}
	if req.GetPatientId() != "" {
		id, err := uuid.Parse(req.GetPatientId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid patient_id")
		}
		filter.PatientID = &id
	}
	if req.GetStaffId() != "" {
		id, err := uuid.Parse(req.GetStaffId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
		}
		filter.StaffID = &id
	}
	if req.GetHospitalId() != "" {
		id, err := uuid.Parse(req.GetHospitalId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
		}
		filter.HospitalID = &id
	}
	list, err := h.appointments.ListAppointments(ctx, actor, filter)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*appointmentpb.Appointment, 0, len(list))
	for i := range list {
		out = append(out, mappers.AppointmentToProto(&list[i]))
	}
	return &appointmentpb.ListAppointmentsResponse{Appointments: out}, nil
}

func (h *AppointmentGRPCHandler) GetAppointment(ctx context.Context, req *appointmentpb.GetAppointmentRequest) (*appointmentpb.GetAppointmentResponse, error) {
	actor, err := actorFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetAppointmentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid appointment_id")
	}
	appt, err := h.appointments.GetAppointment(ctx, actor, id)
	if err != nil {
		return nil, mapError(err)
	}
	return &appointmentpb.GetAppointmentResponse{Appointment: mappers.AppointmentToProto(appt)}, nil
}

func actorFromContext(ctx context.Context) (models.Actor, error) {
	token, ok := grpcclient.ForwardedTokenFromContext(ctx)
	if !ok || token == "" {
		return models.Actor{}, status.Error(codes.Unauthenticated, "missing forwarded token")
	}
	claims, err := sharedauth.VerifyToken(token)
	if err != nil {
		return models.Actor{}, status.Error(codes.Unauthenticated, "invalid token")
	}
	actorID := claims.PatientID
	if claims.ActorType == sharedauth.ActorStaff {
		actorID = claims.StaffID
	}
	if claims.ActorType == sharedauth.ActorAdmin {
		actorID = claims.AdminID
	}
	return models.Actor{
		ActorType:  claims.ActorType,
		ActorID:    actorID,
		PatientID:  claims.PatientID,
		StaffID:    claims.StaffID,
		HospitalID: claims.HospitalID,
	}, nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domainerrors.ErrNotFound), errors.Is(err, domainerrors.ErrNoAvailability):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domainerrors.ErrForbidden), errors.Is(err, domainerrors.ErrConsentRequired):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domainerrors.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domainerrors.ErrConflict), errors.Is(err, domainerrors.ErrSlotUnavailable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domainerrors.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}
