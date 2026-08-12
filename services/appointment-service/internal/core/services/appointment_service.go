package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/outbound"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var bookingTracer = otel.Tracer("appointment-service.booking")

type appointmentService struct {
	appointments outbound.AppointmentRepository
	availability outbound.AvailabilityRepository
	events       outbound.EventPublisher
	livekit      outbound.LiveKitProvider
	audit        outbound.AuditLogger
}

func NewAppointmentService(
	appointments outbound.AppointmentRepository,
	availability outbound.AvailabilityRepository,
	events outbound.EventPublisher,
	livekit outbound.LiveKitProvider,
	audit outbound.AuditLogger,
) inbound.AppointmentService {
	return &appointmentService{
		appointments: appointments,
		availability: availability,
		events:       events,
		livekit:      livekit,
		audit:        audit,
	}
}

func (s *appointmentService) ListAvailableSlots(
	ctx context.Context,
	actor models.Actor,
	staffID, hospitalID uuid.UUID,
	from, to time.Time,
	limit int32,
) ([]models.AppointmentSlot, error) {
	ctx, span := bookingTracer.Start(ctx, "ListAvailableSlots")
	defer span.End()
	span.SetAttributes(
		attribute.String("hospital_id", hospitalID.String()),
		attribute.String("staff_id", staffID.String()),
	)
	if err := s.authorizeSlotRead(actor, staffID, hospitalID); err != nil {
		return nil, err
	}
	return s.computeSlots(ctx, staffID, hospitalID, from, to, int(limit))
}

func (s *appointmentService) GetNextAvailableSlot(
	ctx context.Context,
	actor models.Actor,
	staffID, hospitalID uuid.UUID,
	after time.Time,
) (*models.AppointmentSlot, error) {
	ctx, span := bookingTracer.Start(ctx, "GetNextAvailableSlot")
	defer span.End()
	if after.IsZero() {
		after = time.Now().UTC()
	}
	slots, err := s.ListAvailableSlots(ctx, actor, staffID, hospitalID, after, after.AddDate(0, 0, 28), 1)
	if err != nil {
		return nil, err
	}
	if len(slots) == 0 {
		return nil, domainerrors.ErrNoAvailability
	}
	return &slots[0], nil
}

func (s *appointmentService) BookAppointment(ctx context.Context, actor models.Actor, cmd *models.BookAppointmentCommand) (*models.Appointment, error) {
	ctx, span := bookingTracer.Start(ctx, "BookAppointment")
	defer span.End()
	span.SetAttributes(
		attribute.String("hospital_id", cmd.HospitalID.String()),
		attribute.String("channel", string(cmd.Channel)),
		attribute.String("correlation_id", cmd.CorrelationID.String()),
	)

	if err := s.authorizeBook(ctx, actor, cmd); err != nil {
		s.audit.Log(ctx, sharedaudit.EventAppointmentBookDenied, "denied", cmd.CorrelationID.String(), map[string]any{
			"reason": err.Error(),
		})
		return nil, err
	}
	if cmd.DurationMinutes <= 0 {
		cmd.DurationMinutes = 30
	}
	if cmd.Timezone == "" {
		cmd.Timezone = "UTC"
	}
	if cmd.Type == "" {
		cmd.Type = models.AppointmentTypeVideo
	}
	if cmd.Channel == "" {
		cmd.Channel = models.AppointmentChannelPortal
	}
	if cmd.Title == "" {
		cmd.Title = "Zorba Health appointment"
	}
	if cmd.CorrelationID == uuid.Nil {
		cmd.CorrelationID = uuid.New()
	}
	if !cmd.StartsAt.After(time.Now().UTC()) {
		return nil, fmt.Errorf("%w: starts_at must be in the future", domainerrors.ErrInvalidArgument)
	}

	endsAt := cmd.StartsAt.Add(time.Duration(cmd.DurationMinutes) * time.Minute)
	slots, err := s.computeSlots(ctx, cmd.StaffID, cmd.HospitalID, cmd.StartsAt.Add(-time.Minute), endsAt.Add(time.Minute), 50)
	if err != nil {
		return nil, err
	}
	matched := false
	for _, slot := range slots {
		if slot.StartsAt.Equal(cmd.StartsAt.UTC()) && slot.DurationMinutes == cmd.DurationMinutes {
			matched = true
			break
		}
		// Allow booking if starts_at falls within a matching free slot window of same duration.
		if !cmd.StartsAt.Before(slot.StartsAt) && cmd.StartsAt.Add(time.Duration(cmd.DurationMinutes)*time.Minute).Equal(slot.EndsAt) {
			matched = true
			break
		}
		if slot.StartsAt.Equal(cmd.StartsAt.UTC()) {
			matched = true
			break
		}
	}
	if !matched {
		s.audit.Log(ctx, sharedaudit.EventAppointmentBookDenied, "denied", cmd.CorrelationID.String(), map[string]any{
			"reason": "slot_unavailable",
		})
		return nil, domainerrors.ErrSlotUnavailable
	}

	appt := &models.Appointment{
		ID:                uuid.New(),
		PatientID:         cmd.PatientID,
		StaffID:           cmd.StaffID,
		HospitalID:        cmd.HospitalID,
		StartsAt:          cmd.StartsAt.UTC(),
		EndsAt:            endsAt.UTC(),
		DurationMinutes:   cmd.DurationMinutes,
		Timezone:          cmd.Timezone,
		Type:              cmd.Type,
		Status:            models.AppointmentStatusBooked,
		Channel:           cmd.Channel,
		Title:             cmd.Title,
		Notes:             cmd.Notes,
		CorrelationID:     cmd.CorrelationID,
		VoiceSessionID:    cmd.VoiceSessionID,
		BookedByActorType: cmd.BookedByActorType,
		BookedByActorID:   cmd.BookedByActorID,
	}
	if appt.BookedByActorType == "" {
		appt.BookedByActorType = actor.ActorType
	}
	if appt.BookedByActorID == "" {
		appt.BookedByActorID = actor.ActorID
	}

	if cmd.Type == models.AppointmentTypeVideo && s.livekit != nil {
		ctx, lkSpan := bookingTracer.Start(ctx, "CreateLiveKitRoom")
		room, lkErr := s.livekit.CreateMeetingRoom(ctx, outbound.LiveKitCreateInput{
			RoomName: fmt.Sprintf("appt-%s", appt.ID.String()),
			Title:    appt.Title,
		})
		lkSpan.End()
		if lkErr != nil {
			return nil, fmt.Errorf("livekit room: %w", lkErr)
		}
		appt.LiveKitRoomName = room.RoomName
		appt.LiveKitRoomSID = room.RoomSID
		appt.JoinURL = room.JoinURL
		appt.PatientToken = room.PatientToken
		appt.StaffToken = room.StaffToken
	}

	created, err := s.appointments.Create(ctx, appt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "exclusion") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			return nil, domainerrors.ErrConflict
		}
		return nil, err
	}

	s.audit.Log(ctx, sharedaudit.EventAppointmentBooked, "success", created.CorrelationID.String(), map[string]any{
		"appointment_id": created.ID.String(),
		"hospital_id":    created.HospitalID.String(),
		"channel":        string(created.Channel),
	})

	if s.events != nil {
		patient, _ := s.appointments.GetPatientContact(ctx, created.PatientID)
		staff, _ := s.appointments.GetStaffContact(ctx, created.StaffID)
		hospital, _ := s.appointments.GetHospitalContact(ctx, created.HospitalID)
		_ = s.events.PublishAppointmentBooked(ctx, bookedEventData(created, patient, staff, hospital, cmd.SendSMS, cmd.SendEmail))
	}
	return created, nil
}

func (s *appointmentService) RescheduleAppointment(
	ctx context.Context,
	actor models.Actor,
	appointmentID uuid.UUID,
	startsAt time.Time,
	durationMinutes int32,
	timezone, title string,
) (*models.Appointment, error) {
	ctx, span := bookingTracer.Start(ctx, "RescheduleAppointment")
	defer span.End()

	appt, err := s.appointments.GetByID(ctx, appointmentID)
	if err != nil {
		return nil, err
	}
	if err := s.authorizeAppointmentMutate(actor, appt); err != nil {
		return nil, err
	}
	if appt.Status != models.AppointmentStatusBooked {
		return nil, fmt.Errorf("%w: only booked appointments can be rescheduled", domainerrors.ErrInvalidArgument)
	}
	if durationMinutes <= 0 {
		durationMinutes = appt.DurationMinutes
	}
	if timezone == "" {
		timezone = appt.Timezone
	}
	if title != "" {
		appt.Title = title
	}
	endsAt := startsAt.Add(time.Duration(durationMinutes) * time.Minute)
	slots, err := s.computeSlots(ctx, appt.StaffID, appt.HospitalID, startsAt.Add(-time.Minute), endsAt.Add(time.Minute), 50)
	if err != nil {
		return nil, err
	}
	ok := false
	for _, slot := range slots {
		if slot.StartsAt.Equal(startsAt.UTC()) {
			ok = true
			break
		}
	}
	// Also allow if the only conflict is this appointment itself — re-check after temporarily ignoring it.
	if !ok {
		// Accept if no other booked appointment overlaps and within a rule window by recomputing without this appt.
		booked, bErr := s.appointments.ListBookedOverlapping(ctx, appt.StaffID, startsAt.Add(-24*time.Hour), endsAt.Add(24*time.Hour))
		if bErr != nil {
			return nil, bErr
		}
		filtered := make([]models.Appointment, 0, len(booked))
		for _, b := range booked {
			if b.ID != appt.ID {
				filtered = append(filtered, b)
			}
		}
		rules, _ := s.availability.ListRules(ctx, appt.StaffID, appt.HospitalID)
		exceptions, _ := s.availability.ListExceptions(ctx, appt.StaffID, appt.HospitalID, startsAt.Add(-time.Hour), endsAt.Add(time.Hour))
		recomputed, cErr := ComputeSlots(appt.StaffID, appt.HospitalID, rules, exceptions, filtered, startsAt.Add(-time.Minute), endsAt.Add(time.Minute), 50)
		if cErr != nil {
			return nil, cErr
		}
		for _, slot := range recomputed {
			if slot.StartsAt.Equal(startsAt.UTC()) {
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, domainerrors.ErrSlotUnavailable
	}

	appt.StartsAt = startsAt.UTC()
	appt.EndsAt = endsAt.UTC()
	appt.DurationMinutes = durationMinutes
	appt.Timezone = timezone
	appt.UpdatedAt = time.Now().UTC()

	updated, err := s.appointments.Update(ctx, appt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "exclusion") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			return nil, domainerrors.ErrConflict
		}
		return nil, err
	}
	s.audit.Log(ctx, sharedaudit.EventAppointmentRescheduled, "success", updated.CorrelationID.String(), map[string]any{
		"appointment_id": updated.ID.String(),
	})
	return updated, nil
}

func (s *appointmentService) CancelAppointment(ctx context.Context, actor models.Actor, appointmentID uuid.UUID, reason string) (*models.Appointment, error) {
	ctx, span := bookingTracer.Start(ctx, "CancelAppointment")
	defer span.End()

	appt, err := s.appointments.GetByID(ctx, appointmentID)
	if err != nil {
		return nil, err
	}
	if err := s.authorizeAppointmentMutate(actor, appt); err != nil {
		return nil, err
	}
	if appt.Status == models.AppointmentStatusCancelled {
		return appt, nil
	}
	appt.Status = models.AppointmentStatusCancelled
	appt.UpdatedAt = time.Now().UTC()
	if reason != "" {
		if appt.Notes != "" {
			appt.Notes = appt.Notes + "\n[cancel] " + reason
		} else {
			appt.Notes = "[cancel] " + reason
		}
	}
	updated, err := s.appointments.Update(ctx, appt)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, sharedaudit.EventAppointmentCancelled, "success", updated.CorrelationID.String(), map[string]any{
		"appointment_id": updated.ID.String(),
		"reason":         reason,
	})
	if s.events != nil {
		patient, _ := s.appointments.GetPatientContact(ctx, updated.PatientID)
		staff, _ := s.appointments.GetStaffContact(ctx, updated.StaffID)
		_ = s.events.PublishAppointmentCancelled(ctx, cancelledEventData(updated, patient, staff, reason))
	}
	return updated, nil
}

func (s *appointmentService) ListAppointments(ctx context.Context, actor models.Actor, filter models.ListAppointmentsFilter) ([]models.Appointment, error) {
	adjusted, err := s.authorizeList(actor, filter)
	if err != nil {
		return nil, err
	}
	if adjusted.Limit <= 0 {
		adjusted.Limit = 50
	}
	return s.appointments.List(ctx, adjusted)
}

func (s *appointmentService) GetAppointment(ctx context.Context, actor models.Actor, appointmentID uuid.UUID) (*models.Appointment, error) {
	appt, err := s.appointments.GetByID(ctx, appointmentID)
	if err != nil {
		return nil, err
	}
	if err := s.authorizeAppointmentRead(actor, appt); err != nil {
		return nil, err
	}
	return appt, nil
}

func (s *appointmentService) computeSlots(ctx context.Context, staffID, hospitalID uuid.UUID, from, to time.Time, limit int) ([]models.AppointmentSlot, error) {
	ctx, span := bookingTracer.Start(ctx, "ComputeSlots")
	defer span.End()

	rules, err := s.availability.ListRules(ctx, staffID, hospitalID)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return []models.AppointmentSlot{}, nil
	}
	exceptions, err := s.availability.ListExceptions(ctx, staffID, hospitalID, from, to)
	if err != nil {
		return nil, err
	}
	booked, err := s.appointments.ListBookedOverlapping(ctx, staffID, from, to)
	if err != nil {
		return nil, err
	}
	return ComputeSlots(staffID, hospitalID, rules, exceptions, booked, from, to, limit)
}

func (s *appointmentService) authorizeSlotRead(actor models.Actor, staffID, hospitalID uuid.UUID) error {
	_ = staffID
	switch actor.ActorType {
	case sharedauth.ActorPatient, sharedauth.ActorStaff, sharedauth.ActorAdmin:
		if actor.ActorType == sharedauth.ActorStaff && actor.HospitalID != "" && actor.HospitalID != hospitalID.String() {
			return domainerrors.ErrForbidden
		}
		return nil
	default:
		return domainerrors.ErrUnauthorized
	}
}

func (s *appointmentService) authorizeBook(ctx context.Context, actor models.Actor, cmd *models.BookAppointmentCommand) error {
	switch actor.ActorType {
	case sharedauth.ActorPatient:
		if actor.PatientID == "" || actor.PatientID != cmd.PatientID.String() {
			return domainerrors.ErrForbidden
		}
	case sharedauth.ActorStaff:
		if actor.StaffID == "" {
			return domainerrors.ErrForbidden
		}
		if actor.HospitalID != "" && actor.HospitalID != cmd.HospitalID.String() {
			return domainerrors.ErrForbidden
		}
		// Staff may book for any patient with active consent at their hospital.
	case sharedauth.ActorAdmin:
		// allowed
	default:
		return domainerrors.ErrUnauthorized
	}
	ok, err := s.appointments.HasActiveHospitalConsent(ctx, cmd.PatientID, cmd.HospitalID)
	if err != nil {
		return err
	}
	if !ok {
		return domainerrors.ErrConsentRequired
	}
	return nil
}

func (s *appointmentService) authorizeAppointmentRead(actor models.Actor, appt *models.Appointment) error {
	switch actor.ActorType {
	case sharedauth.ActorPatient:
		if actor.PatientID != appt.PatientID.String() {
			return domainerrors.ErrForbidden
		}
	case sharedauth.ActorStaff:
		if actor.HospitalID != "" && actor.HospitalID != appt.HospitalID.String() {
			return domainerrors.ErrForbidden
		}
	case sharedauth.ActorAdmin:
		return nil
	default:
		return domainerrors.ErrUnauthorized
	}
	return nil
}

func (s *appointmentService) authorizeAppointmentMutate(actor models.Actor, appt *models.Appointment) error {
	return s.authorizeAppointmentRead(actor, appt)
}

func (s *appointmentService) authorizeList(actor models.Actor, filter models.ListAppointmentsFilter) (models.ListAppointmentsFilter, error) {
	switch actor.ActorType {
	case sharedauth.ActorPatient:
		if filter.PatientID == nil || filter.PatientID.String() != actor.PatientID {
			return filter, domainerrors.ErrForbidden
		}
	case sharedauth.ActorStaff:
		if actor.HospitalID != "" {
			hid, err := uuid.Parse(actor.HospitalID)
			if err != nil {
				return filter, domainerrors.ErrForbidden
			}
			if filter.HospitalID == nil {
				filter.HospitalID = &hid
			} else if filter.HospitalID.String() != actor.HospitalID {
				return filter, domainerrors.ErrForbidden
			}
		}
	case sharedauth.ActorAdmin:
		return filter, nil
	default:
		return filter, domainerrors.ErrUnauthorized
	}
	return filter, nil
}
