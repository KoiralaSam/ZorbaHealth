package inbound

import (
	"context"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/google/uuid"
)

type AvailabilityService interface {
	SetAvailabilityRules(ctx context.Context, actor models.Actor, staffID, hospitalID uuid.UUID, rules []models.AvailabilityRule) ([]models.AvailabilityRule, error)
	GetAvailabilityRules(ctx context.Context, actor models.Actor, staffID, hospitalID uuid.UUID) ([]models.AvailabilityRule, error)
	AddAvailabilityException(ctx context.Context, actor models.Actor, ex models.AvailabilityException) (*models.AvailabilityException, error)
	RemoveAvailabilityException(ctx context.Context, actor models.Actor, exceptionID uuid.UUID) error
	ListAvailabilityExceptions(ctx context.Context, actor models.Actor, staffID, hospitalID uuid.UUID, from, to time.Time) ([]models.AvailabilityException, error)
}

type AppointmentService interface {
	ListAvailableSlots(ctx context.Context, actor models.Actor, staffID, hospitalID uuid.UUID, from, to time.Time, limit int32) ([]models.AppointmentSlot, error)
	GetNextAvailableSlot(ctx context.Context, actor models.Actor, staffID, hospitalID uuid.UUID, after time.Time) (*models.AppointmentSlot, error)
	BookAppointment(ctx context.Context, actor models.Actor, cmd *models.BookAppointmentCommand) (*models.Appointment, error)
	RescheduleAppointment(ctx context.Context, actor models.Actor, appointmentID uuid.UUID, startsAt time.Time, durationMinutes int32, timezone, title string) (*models.Appointment, error)
	CancelAppointment(ctx context.Context, actor models.Actor, appointmentID uuid.UUID, reason string) (*models.Appointment, error)
	ListAppointments(ctx context.Context, actor models.Actor, filter models.ListAppointmentsFilter) ([]models.Appointment, error)
	GetAppointment(ctx context.Context, actor models.Actor, appointmentID uuid.UUID) (*models.Appointment, error)
}
