package outbound

import (
	"context"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/google/uuid"
)

type AvailabilityRepository interface {
	ReplaceRules(ctx context.Context, staffID, hospitalID uuid.UUID, rules []models.AvailabilityRule) ([]models.AvailabilityRule, error)
	ListRules(ctx context.Context, staffID, hospitalID uuid.UUID) ([]models.AvailabilityRule, error)
	AddException(ctx context.Context, ex models.AvailabilityException) (*models.AvailabilityException, error)
	RemoveException(ctx context.Context, exceptionID uuid.UUID) error
	GetException(ctx context.Context, exceptionID uuid.UUID) (*models.AvailabilityException, error)
	ListExceptions(ctx context.Context, staffID, hospitalID uuid.UUID, from, to time.Time) ([]models.AvailabilityException, error)
}

type AppointmentRepository interface {
	Create(ctx context.Context, appt *models.Appointment) (*models.Appointment, error)
	Update(ctx context.Context, appt *models.Appointment) (*models.Appointment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Appointment, error)
	List(ctx context.Context, filter models.ListAppointmentsFilter) ([]models.Appointment, error)
	ListBookedOverlapping(ctx context.Context, staffID uuid.UUID, from, to time.Time) ([]models.Appointment, error)
	GetPatientContact(ctx context.Context, patientID uuid.UUID) (*models.PatientContact, error)
	GetStaffContact(ctx context.Context, staffID uuid.UUID) (*models.StaffContact, error)
	GetHospitalContact(ctx context.Context, hospitalID uuid.UUID) (*models.HospitalContact, error)
	HasActiveHospitalConsent(ctx context.Context, patientID, hospitalID uuid.UUID) (bool, error)
}

type EventPublisher interface {
	PublishAppointmentBooked(ctx context.Context, data *events.AppointmentBookedData) error
	PublishAppointmentCancelled(ctx context.Context, data *events.AppointmentCancelledData) error
}

type LiveKitCreateInput struct {
	RoomName     string
	Title        string
	EmptyTimeout uint32
}

type LiveKitCreateResult struct {
	RoomName     string
	RoomSID      string
	JoinURL      string
	PatientToken string
	StaffToken   string
}

type LiveKitProvider interface {
	CreateMeetingRoom(ctx context.Context, in LiveKitCreateInput) (*LiveKitCreateResult, error)
}

type AuditLogger interface {
	Log(ctx context.Context, eventType, outcome, correlationID string, attrs map[string]any)
}
