package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type AppointmentType string

const (
	AppointmentTypeVideo    AppointmentType = "video"
	AppointmentTypeInPerson AppointmentType = "in_person"
)

type AppointmentStatus string

const (
	AppointmentStatusBooked     AppointmentStatus = "booked"
	AppointmentStatusCancelled  AppointmentStatus = "cancelled"
	AppointmentStatusCompleted  AppointmentStatus = "completed"
	AppointmentStatusNoShow     AppointmentStatus = "no_show"
)

type AppointmentChannel string

const (
	AppointmentChannelVoice     AppointmentChannel = "voice"
	AppointmentChannelPortal    AppointmentChannel = "portal"
	AppointmentChannelMobile    AppointmentChannel = "mobile"
	AppointmentChannelDashboard AppointmentChannel = "dashboard"
)

type AvailabilityRule struct {
	ID                   uuid.UUID
	StaffID              uuid.UUID
	HospitalID           uuid.UUID
	Weekday              int // 0=Sunday .. 6=Saturday
	StartTimeLocal       string // HH:MM
	EndTimeLocal         string // HH:MM
	SlotDurationMinutes  int32
	Timezone             string
	EffectiveFrom        time.Time
	EffectiveUntil       *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type AvailabilityException struct {
	ID          uuid.UUID
	StaffID     uuid.UUID
	HospitalID  uuid.UUID
	StartsAt    time.Time
	EndsAt      time.Time
	Reason      string
	IsAvailable bool
	CreatedAt   time.Time
}

type AppointmentSlot struct {
	StartsAt        time.Time
	EndsAt          time.Time
	DurationMinutes int32
	Timezone        string
	StaffID         uuid.UUID
	HospitalID      uuid.UUID
}

type Appointment struct {
	ID                 uuid.UUID
	PatientID          uuid.UUID
	StaffID            uuid.UUID
	HospitalID         uuid.UUID
	StartsAt           time.Time
	EndsAt             time.Time
	DurationMinutes    int32
	Timezone           string
	Type               AppointmentType
	Status             AppointmentStatus
	Channel            AppointmentChannel
	Title              string
	Notes              string
	CorrelationID      uuid.UUID
	VoiceSessionID     string
	BookedByActorType  string
	BookedByActorID    string
	JoinURL            string
	LiveKitRoomName    string
	LiveKitRoomSID     string
	PatientToken       string
	StaffToken         string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type BookAppointmentCommand struct {
	PatientID         uuid.UUID
	StaffID           uuid.UUID
	HospitalID        uuid.UUID
	StartsAt          time.Time
	DurationMinutes   int32
	Timezone          string
	Type              AppointmentType
	Channel           AppointmentChannel
	Title             string
	Notes             string
	CorrelationID     uuid.UUID
	VoiceSessionID    string
	BookedByActorType string
	BookedByActorID   string
	SendSMS           bool
	SendEmail         bool
}

type ListAppointmentsFilter struct {
	PatientID        *uuid.UUID
	StaffID          *uuid.UUID
	HospitalID       *uuid.UUID
	IncludeCancelled bool
	Limit            int32
}

type Actor struct {
	ActorType  string
	ActorID    string
	PatientID  string
	StaffID    string
	HospitalID string
}

type PatientContact struct {
	ID          uuid.UUID
	Email       string
	PhoneNumber string
	FullName    string
}

type StaffContact struct {
	ID          uuid.UUID
	HospitalID  uuid.UUID
	Email       string
	Name        string
	PhoneNumber string
}

type HospitalContact struct {
	ID         uuid.UUID
	Name       string
	Address    string
	City       string
	State      string
	PostalCode string
	Phone      string
}

// FormattedAddress returns a single-line mailing address when present.
func (h *HospitalContact) FormattedAddress() string {
	if h == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if h.Address != "" {
		parts = append(parts, h.Address)
	}
	cityState := strings.TrimSpace(strings.TrimSpace(h.City) + " " + strings.TrimSpace(h.State))
	if cityState != "" {
		parts = append(parts, cityState)
	}
	if h.PostalCode != "" {
		parts = append(parts, h.PostalCode)
	}
	return strings.Join(parts, ", ")
}

// MapsQuery prefers a full address; falls back to hospital name for Google Maps search.
func (h *HospitalContact) MapsQuery() string {
	if h == nil {
		return ""
	}
	if addr := h.FormattedAddress(); addr != "" {
		if h.Name != "" {
			return h.Name + ", " + addr
		}
		return addr
	}
	return h.Name
}
