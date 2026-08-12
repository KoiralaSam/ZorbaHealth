package services

import (
	"testing"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/models"
	"github.com/google/uuid"
)

func TestComputeSlots_BasicWeekdayWindow(t *testing.T) {
	staffID := uuid.New()
	hospitalID := uuid.New()
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	// Pick a known Monday in the future relative to "now" inside ComputeSlots.
	// Use a wide window far ahead so slots are not filtered as past.
	from := time.Date(2099, 3, 2, 0, 0, 0, 0, loc) // Monday
	to := from.AddDate(0, 0, 1)

	rules := []models.AvailabilityRule{{
		StaffID:             staffID,
		HospitalID:          hospitalID,
		Weekday:             int(time.Monday),
		StartTimeLocal:      "09:00",
		EndTimeLocal:        "11:00",
		SlotDurationMinutes: 30,
		Timezone:            "America/New_York",
		EffectiveFrom:       from.AddDate(0, 0, -7),
	}}

	slots, err := ComputeSlots(staffID, hospitalID, rules, nil, nil, from.UTC(), to.UTC(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 4 { // 09:00, 09:30, 10:00, 10:30
		t.Fatalf("expected 4 slots, got %d: %+v", len(slots), slots)
	}
	if slots[0].DurationMinutes != 30 {
		t.Fatalf("unexpected duration %d", slots[0].DurationMinutes)
	}
}

func TestComputeSlots_SubtractsBookedAndExceptions(t *testing.T) {
	staffID := uuid.New()
	hospitalID := uuid.New()
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2099, 6, 1, 0, 0, 0, 0, loc) // Monday
	to := from.AddDate(0, 0, 1)

	rules := []models.AvailabilityRule{{
		StaffID:             staffID,
		HospitalID:          hospitalID,
		Weekday:             int(time.Monday),
		StartTimeLocal:      "09:00",
		EndTimeLocal:        "12:00",
		SlotDurationMinutes: 60,
		Timezone:            "UTC",
		EffectiveFrom:       from.AddDate(0, 0, -1),
	}}

	bookedStart := time.Date(2099, 6, 1, 10, 0, 0, 0, loc)
	booked := []models.Appointment{{
		ID:        uuid.New(),
		StaffID:   staffID,
		StartsAt:  bookedStart,
		EndsAt:    bookedStart.Add(time.Hour),
		Status:    models.AppointmentStatusBooked,
	}}

	exStart := time.Date(2099, 6, 1, 9, 0, 0, 0, loc)
	exceptions := []models.AvailabilityException{{
		StaffID:     staffID,
		HospitalID:  hospitalID,
		StartsAt:    exStart,
		EndsAt:      exStart.Add(time.Hour),
		IsAvailable: false,
	}}

	slots, err := ComputeSlots(staffID, hospitalID, rules, exceptions, booked, from.UTC(), to.UTC(), 20)
	if err != nil {
		t.Fatal(err)
	}
	// 09:00 blocked by exception, 10:00 booked, only 11:00 remains.
	if len(slots) != 1 {
		t.Fatalf("expected 1 slot, got %d: %+v", len(slots), slots)
	}
	if !slots[0].StartsAt.Equal(time.Date(2099, 6, 1, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected slot start %s", slots[0].StartsAt)
	}
}

func TestComputeSlots_InvalidRange(t *testing.T) {
	staffID := uuid.New()
	hospitalID := uuid.New()
	now := time.Now().UTC()
	_, err := ComputeSlots(staffID, hospitalID, nil, nil, nil, now, now, 10)
	if err == nil {
		t.Fatal("expected error for non-increasing range")
	}
}

func TestAuthorizeBookPatientMismatch(t *testing.T) {
	svc := &appointmentService{}
	actor := models.Actor{ActorType: "patient", PatientID: "p1"}
	cmd := &models.BookAppointmentCommand{PatientID: uuid.New()}
	err := svc.authorizeBook(nil, actor, cmd)
	if err == nil {
		t.Fatal("expected forbidden for mismatched patient")
	}
}

func TestAuthorizeListPatientRequiresOwnFilter(t *testing.T) {
	svc := &appointmentService{}
	pid := uuid.New()
	actor := models.Actor{ActorType: "patient", PatientID: pid.String()}
	_, err := svc.authorizeList(actor, models.ListAppointmentsFilter{})
	if err == nil {
		t.Fatal("expected forbidden when patient filter missing")
	}
	_, err = svc.authorizeList(actor, models.ListAppointmentsFilter{PatientID: &pid})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
