package services

import (
	"context"
	"fmt"
	"net/url"
	"time"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/outbound"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/google/uuid"
)

type availabilityService struct {
	repo  outbound.AvailabilityRepository
	audit outbound.AuditLogger
}

func NewAvailabilityService(repo outbound.AvailabilityRepository, audit outbound.AuditLogger) inbound.AvailabilityService {
	return &availabilityService{repo: repo, audit: audit}
}

func (s *availabilityService) SetAvailabilityRules(
	ctx context.Context,
	actor models.Actor,
	staffID, hospitalID uuid.UUID,
	rules []models.AvailabilityRule,
) ([]models.AvailabilityRule, error) {
	if err := authorizeStaffHospital(actor, staffID, hospitalID); err != nil {
		return nil, err
	}
	for i := range rules {
		rules[i].StaffID = staffID
		rules[i].HospitalID = hospitalID
		if rules[i].SlotDurationMinutes <= 0 {
			rules[i].SlotDurationMinutes = 30
		}
		if rules[i].Timezone == "" {
			rules[i].Timezone = "UTC"
		}
		if rules[i].Weekday < 0 || rules[i].Weekday > 6 {
			return nil, fmt.Errorf("%w: weekday must be 0-6", domainerrors.ErrInvalidArgument)
		}
	}
	out, err := s.repo.ReplaceRules(ctx, staffID, hospitalID, rules)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, sharedaudit.EventAvailabilityUpdated, "success", "", map[string]any{
		"staff_id":     staffID.String(),
		"hospital_id":  hospitalID.String(),
		"rules_count":  len(out),
	})
	return out, nil
}

func (s *availabilityService) GetAvailabilityRules(
	ctx context.Context,
	actor models.Actor,
	staffID, hospitalID uuid.UUID,
) ([]models.AvailabilityRule, error) {
	if err := authorizeStaffHospitalOrPatient(actor, hospitalID); err != nil {
		return nil, err
	}
	return s.repo.ListRules(ctx, staffID, hospitalID)
}

func (s *availabilityService) AddAvailabilityException(
	ctx context.Context,
	actor models.Actor,
	ex models.AvailabilityException,
) (*models.AvailabilityException, error) {
	if err := authorizeStaffHospital(actor, ex.StaffID, ex.HospitalID); err != nil {
		return nil, err
	}
	if !ex.EndsAt.After(ex.StartsAt) {
		return nil, fmt.Errorf("%w: ends_at must be after starts_at", domainerrors.ErrInvalidArgument)
	}
	created, err := s.repo.AddException(ctx, ex)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, sharedaudit.EventAvailabilityUpdated, "success", "", map[string]any{
		"staff_id":      ex.StaffID.String(),
		"hospital_id":   ex.HospitalID.String(),
		"exception_id":  created.ID.String(),
		"is_available":  created.IsAvailable,
	})
	return created, nil
}

func (s *availabilityService) RemoveAvailabilityException(ctx context.Context, actor models.Actor, exceptionID uuid.UUID) error {
	ex, err := s.repo.GetException(ctx, exceptionID)
	if err != nil {
		return err
	}
	if err := authorizeStaffHospital(actor, ex.StaffID, ex.HospitalID); err != nil {
		return err
	}
	if err := s.repo.RemoveException(ctx, exceptionID); err != nil {
		return err
	}
	s.audit.Log(ctx, sharedaudit.EventAvailabilityUpdated, "success", "", map[string]any{
		"exception_id": exceptionID.String(),
		"action":       "remove_exception",
	})
	return nil
}

func (s *availabilityService) ListAvailabilityExceptions(
	ctx context.Context,
	actor models.Actor,
	staffID, hospitalID uuid.UUID,
	from, to time.Time,
) ([]models.AvailabilityException, error) {
	if err := authorizeStaffHospitalOrPatient(actor, hospitalID); err != nil {
		return nil, err
	}
	return s.repo.ListExceptions(ctx, staffID, hospitalID, from, to)
}

func authorizeStaffHospital(actor models.Actor, staffID, hospitalID uuid.UUID) error {
	_ = staffID
	switch actor.ActorType {
	case sharedauth.ActorStaff:
		if actor.HospitalID != "" && actor.HospitalID != hospitalID.String() {
			return domainerrors.ErrForbidden
		}
		// Prefer staff manage their own calendar; hospital admins may manage any staff in hospital.
		if actor.StaffID != "" && actor.StaffID != staffID.String() {
			// Allow — hospital staff dashboards may set availability for any staff in hospital.
		}
		return nil
	case sharedauth.ActorAdmin:
		return nil
	default:
		return domainerrors.ErrForbidden
	}
}

func authorizeStaffHospitalOrPatient(actor models.Actor, hospitalID uuid.UUID) error {
	switch actor.ActorType {
	case sharedauth.ActorPatient:
		return nil
	case sharedauth.ActorStaff:
		if actor.HospitalID != "" && actor.HospitalID != hospitalID.String() {
			return domainerrors.ErrForbidden
		}
		return nil
	case sharedauth.ActorAdmin:
		return nil
	default:
		return domainerrors.ErrUnauthorized
	}
}

func bookedEventData(
	appt *models.Appointment,
	patient *models.PatientContact,
	staff *models.StaffContact,
	hospital *models.HospitalContact,
	sendSMS, sendEmail bool,
) *events.AppointmentBookedData {
	data := &events.AppointmentBookedData{
		AppointmentID:   appt.ID.String(),
		PatientID:       appt.PatientID.String(),
		StaffID:         appt.StaffID.String(),
		HospitalID:      appt.HospitalID.String(),
		CorrelationID:   appt.CorrelationID.String(),
		StartsAtRFC3339: appt.StartsAt.Format(time.RFC3339),
		DurationMinutes: int(appt.DurationMinutes),
		Timezone:        appt.Timezone,
		Title:           appt.Title,
		Type:            string(appt.Type),
		Channel:         string(appt.Channel),
		JoinURL:         appt.JoinURL,
		SendSMS:         sendSMS,
		SendEmail:       sendEmail,
		LiveKitRoomName: appt.LiveKitRoomName,
		PatientToken:    appt.PatientToken,
		StaffToken:      appt.StaffToken,
	}
	if patient != nil {
		data.PatientEmail = patient.Email
		data.PatientPhone = patient.PhoneNumber
		data.PatientName = patient.FullName
	}
	if staff != nil {
		data.StaffEmail = staff.Email
		data.StaffName = staff.Name
		data.StaffPhone = staff.PhoneNumber
	}
	if hospital != nil {
		data.HospitalName = hospital.Name
		data.HospitalAddress = hospital.FormattedAddress()
		data.HospitalPhone = hospital.Phone
		if q := hospital.MapsQuery(); q != "" {
			data.MapsURL = "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(q)
		}
	}
	return data
}

func cancelledEventData(appt *models.Appointment, patient *models.PatientContact, staff *models.StaffContact, reason string) *events.AppointmentCancelledData {
	data := &events.AppointmentCancelledData{
		AppointmentID:   appt.ID.String(),
		PatientID:       appt.PatientID.String(),
		StaffID:         appt.StaffID.String(),
		HospitalID:      appt.HospitalID.String(),
		CorrelationID:   appt.CorrelationID.String(),
		StartsAtRFC3339: appt.StartsAt.Format(time.RFC3339),
		DurationMinutes: int(appt.DurationMinutes),
		Timezone:        appt.Timezone,
		Title:           appt.Title,
		Type:            string(appt.Type),
		Reason:          reason,
	}
	if patient != nil {
		data.PatientEmail = patient.Email
		data.PatientName = patient.FullName
	}
	if staff != nil {
		data.StaffEmail = staff.Email
		data.StaffName = staff.Name
	}
	return data
}
