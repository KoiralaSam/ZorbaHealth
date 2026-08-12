package services

import (
	"context"
	"testing"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/google/uuid"
)

func TestScheduleHealthStaffMeetingCreatesPendingWithoutLiveKit(t *testing.T) {
	ids := testSchedulingIDs()
	repo := newSchedulingRepo(ids)
	patients := &fakeSchedulingPatientRepo{patient: &models.Patient{ID: ids.patientID, Email: "patient@example.com", FullName: "Pat Patient"}}
	livekit := &fakeLiveKitProvider{}
	publisher := &fakeSchedulingPublisher{}
	svc := NewSchedulingService(repo, patients, nil, livekit, publisher, nil)

	meeting, err := svc.ScheduleHealthStaffMeeting(context.Background(), &models.ScheduleMeetingCommand{
		PatientID:       ids.patientID,
		StaffID:         ids.staffID,
		HospitalID:      ids.hospitalID,
		StartsAt:        time.Now().UTC().Add(2 * time.Hour),
		DurationMinutes: 30,
		Timezone:        "America/Chicago",
		Channel:         models.MeetingChannelVoice,
		CorrelationID:   uuid.New(),
		SendSMS:         true,
		ActorType:       "patient",
		ActorID:         ids.patientID.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if meeting.Status != models.MeetingStatusPending {
		t.Fatalf("status = %s, want pending", meeting.Status)
	}
	if meeting.JoinURL != "" || meeting.LiveKitRoomName != "" {
		t.Fatalf("pending meeting should not have LiveKit join fields: %+v", meeting)
	}
	if livekit.lastInput.RoomName != "" {
		t.Fatalf("livekit room = %q, want no room before acceptance", livekit.lastInput.RoomName)
	}
	if publisher.requested == nil {
		t.Fatal("expected meeting requested event")
	}
	if publisher.scheduled != nil {
		t.Fatal("did not expect scheduled event before staff acceptance")
	}
}

func TestScheduleHealthStaffMeetingByStaffAutoConfirmsWithLiveKit(t *testing.T) {
	ids := testSchedulingIDs()
	repo := newSchedulingRepo(ids)
	patients := &fakeSchedulingPatientRepo{patient: &models.Patient{ID: ids.patientID, Email: "patient@example.com", PhoneNumber: "+15551234567", FullName: "Pat Patient"}}
	livekit := &fakeLiveKitProvider{}
	publisher := &fakeSchedulingPublisher{}
	svc := NewSchedulingService(repo, patients, nil, livekit, publisher, nil)

	meeting, err := svc.ScheduleHealthStaffMeeting(context.Background(), &models.ScheduleMeetingCommand{
		PatientID:       ids.patientID,
		StaffID:         ids.staffID,
		HospitalID:      ids.hospitalID,
		StartsAt:        time.Now().UTC().Add(2 * time.Hour),
		DurationMinutes: 30,
		Timezone:        "America/Chicago",
		Channel:         models.MeetingChannelDashboard,
		CorrelationID:   uuid.New(),
		SendSMS:         true,
		ActorType:       "staff",
		ActorID:         ids.staffID.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if meeting.Status != models.MeetingStatusScheduled {
		t.Fatalf("status = %s, want scheduled", meeting.Status)
	}
	if meeting.JoinURL == "" || meeting.LiveKitRoomName == "" || meeting.PatientToken == "" || meeting.StaffToken == "" {
		t.Fatalf("auto-confirmed meeting missing LiveKit fields: %+v", meeting)
	}
	if livekit.lastInput.RoomName == "" {
		t.Fatal("expected livekit room creation on staff schedule")
	}
	if publisher.requested != nil {
		t.Fatal("did not expect meeting requested event for staff-created visit")
	}
	if publisher.scheduled == nil || publisher.scheduled.JoinURL == "" {
		t.Fatalf("expected scheduled event with join URL, got %+v", publisher.scheduled)
	}
}

func TestAcceptScheduledMeetingCreatesLiveKitRoomAndPublishesScheduled(t *testing.T) {
	ids := testSchedulingIDs()
	repo := newSchedulingRepo(ids)
	existing := repo.pendingMeeting(ids)
	repo.meeting = existing
	patients := &fakeSchedulingPatientRepo{patient: &models.Patient{ID: ids.patientID, Email: "patient@example.com", PhoneNumber: "+15551234567", FullName: "Pat Patient"}}
	livekit := &fakeLiveKitProvider{}
	publisher := &fakeSchedulingPublisher{}
	svc := NewSchedulingService(repo, patients, nil, livekit, publisher, nil)

	meeting, err := svc.AcceptScheduledMeeting(context.Background(), existing.ID.String(), models.ScheduleActor{
		ActorType:  "staff",
		ActorID:    ids.staffID.String(),
		StaffID:    ids.staffID.String(),
		HospitalID: ids.hospitalID.String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if meeting.Status != models.MeetingStatusScheduled {
		t.Fatalf("status = %s, want scheduled", meeting.Status)
	}
	if livekit.lastInput.RoomName == "" {
		t.Fatal("expected livekit room creation")
	}
	if publisher.scheduled == nil || publisher.scheduled.JoinURL == "" {
		t.Fatalf("expected scheduled event with join URL, got %+v", publisher.scheduled)
	}
	if publisher.scheduled.SendSMS != true {
		t.Fatalf("scheduled SendSMS = false, want true")
	}
}

func TestAcceptScheduledMeetingRejectsPatientActor(t *testing.T) {
	ids := testSchedulingIDs()
	repo := newSchedulingRepo(ids)
	existing := repo.pendingMeeting(ids)
	repo.meeting = existing
	svc := NewSchedulingService(repo, &fakeSchedulingPatientRepo{}, nil, &fakeLiveKitProvider{}, &fakeSchedulingPublisher{}, nil)

	_, err := svc.AcceptScheduledMeeting(context.Background(), existing.ID.String(), models.ScheduleActor{
		ActorType: "patient",
		PatientID: ids.patientID.String(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err != domainErrors.ErrMeetingNotFound {
		t.Fatalf("err = %v, want ErrMeetingNotFound", err)
	}
}

func TestRescheduleScheduledMeetingUpdatesTimeBeforeLiveKit(t *testing.T) {
	ids := testSchedulingIDs()
	repo := newSchedulingRepo(ids)
	existing := repo.pendingMeeting(ids)
	repo.meeting = existing
	patients := &fakeSchedulingPatientRepo{patient: &models.Patient{ID: ids.patientID, Email: "patient@example.com", FullName: "Pat Patient"}}
	livekit := &fakeLiveKitProvider{}
	svc := NewSchedulingService(repo, patients, nil, livekit, &fakeSchedulingPublisher{}, nil)
	newStart := time.Now().UTC().Add(4 * time.Hour).Truncate(time.Second)

	meeting, err := svc.RescheduleScheduledMeeting(context.Background(), existing.ID.String(), models.ScheduleActor{
		ActorType:  "staff",
		ActorID:    ids.staffID.String(),
		StaffID:    ids.staffID.String(),
		HospitalID: ids.hospitalID.String(),
	}, newStart, 45, "America/New_York", "Updated title")
	if err != nil {
		t.Fatal(err)
	}
	if !meeting.StartsAt.Equal(newStart) || meeting.DurationMinutes != 45 || meeting.Timezone != "America/New_York" || meeting.Title != "Updated title" {
		t.Fatalf("rescheduled meeting mismatch: %+v", meeting)
	}
	if livekit.lastInput.StartsAtEpoch != newStart.Unix() {
		t.Fatalf("livekit starts_at = %d, want %d", livekit.lastInput.StartsAtEpoch, newStart.Unix())
	}
}

type schedulingIDs struct {
	patientID  uuid.UUID
	staffID    uuid.UUID
	hospitalID uuid.UUID
}

func testSchedulingIDs() schedulingIDs {
	return schedulingIDs{
		patientID:  uuid.New(),
		staffID:    uuid.New(),
		hospitalID: uuid.New(),
	}
}

type fakeMeetingRepo struct {
	ids     schedulingIDs
	staff   *models.StaffSummary
	meeting *models.ScheduledMeeting
}

type fakeLiveKitProvider struct {
	result    *outbound.LiveKitCreateResult
	lastInput outbound.LiveKitCreateInput
}

func (f *fakeLiveKitProvider) MintRoomJoinToken(_ context.Context, roomName, identity string) (*outbound.LiveKitRoomToken, error) {
	return &outbound.LiveKitRoomToken{
		Token: "fake-join-" + identity + "-" + roomName,
		WSURL: "wss://livekit.example",
	}, nil
}

func (f *fakeLiveKitProvider) ResolveRoomName(_ context.Context, value string) (string, error) {
	return value, nil
}

func (f *fakeLiveKitProvider) DialSIPParticipant(_ context.Context, in outbound.DialSIPParticipantInput) (*outbound.DialSIPParticipantResult, error) {
	identity := in.ParticipantIdentity
	if identity == "" {
		identity = "staff-sip-fake"
	}
	return &outbound.DialSIPParticipantResult{
		SIPCallID:           "fake-sip",
		ParticipantID:       "fake-participant",
		ParticipantIdentity: identity,
	}, nil
}

func (f *fakeLiveKitProvider) RemintMeetingTokens(_ context.Context, roomName string, _ time.Duration) (string, string, error) {
	return "remint-patient-" + roomName, "remint-staff-" + roomName, nil
}

func (f *fakeLiveKitProvider) CreateMeetingRoom(ctx context.Context, in outbound.LiveKitCreateInput) (*outbound.LiveKitCreateResult, error) {
	f.lastInput = in
	if f.result != nil {
		return f.result, nil
	}
	return &outbound.LiveKitCreateResult{
		RoomName:     in.RoomName,
		RoomSID:      "stub-room",
		JoinURL:      "wss://livekit.example?room=" + in.RoomName,
		PatientToken: "patient-token",
		StaffToken:   "staff-token",
	}, nil
}

func newSchedulingRepo(ids schedulingIDs) *fakeMeetingRepo {
	return &fakeMeetingRepo{
		ids: ids,
		staff: &models.StaffSummary{
			StaffID:    ids.staffID,
			HospitalID: ids.hospitalID,
			Name:       "Dr Staff",
			Role:       "doctor",
			Email:      "staff@example.com",
		},
	}
}

func (r *fakeMeetingRepo) pendingMeeting(ids schedulingIDs) *models.ScheduledMeeting {
	return &models.ScheduledMeeting{
		ID:                 uuid.New(),
		PatientID:          ids.patientID,
		StaffID:            ids.staffID,
		HospitalID:         ids.hospitalID,
		CreatedByActorType: "patient",
		CreatedByActorID:   ids.patientID.String(),
		StartsAt:           time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second),
		DurationMinutes:    30,
		Timezone:           "America/Chicago",
		Title:              "Zorba Health video visit",
		Status:             models.MeetingStatusPending,
		CorrelationID:      uuid.New(),
		SendSMS:            true,
		Channel:            models.MeetingChannelPortal,
	}
}

func (r *fakeMeetingRepo) Insert(ctx context.Context, meeting *models.ScheduledMeeting) (*models.ScheduledMeeting, error) {
	out := *meeting
	out.ID = uuid.New()
	out.CreatedAt = time.Now().UTC()
	r.meeting = &out
	return &out, nil
}

func (r *fakeMeetingRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.ScheduledMeeting, error) {
	if r.meeting == nil || r.meeting.ID != id {
		return nil, domainErrors.ErrMeetingNotFound
	}
	out := *r.meeting
	return &out, nil
}

func (r *fakeMeetingRepo) List(ctx context.Context, filter models.ListMeetingsFilter) ([]models.ScheduledMeeting, error) {
	if r.meeting == nil {
		return nil, nil
	}
	return []models.ScheduledMeeting{*r.meeting}, nil
}

func (r *fakeMeetingRepo) MarkScheduled(ctx context.Context, meeting *models.ScheduledMeeting) (*models.ScheduledMeeting, error) {
	if r.meeting == nil || r.meeting.ID != meeting.ID || r.meeting.Status != models.MeetingStatusPending {
		return nil, domainErrors.ErrMeetingNotPending
	}
	out := *meeting
	out.Status = models.MeetingStatusScheduled
	r.meeting = &out
	return &out, nil
}

func (r *fakeMeetingRepo) Cancel(ctx context.Context, id uuid.UUID) (*models.ScheduledMeeting, error) {
	if r.meeting == nil || r.meeting.ID != id {
		return nil, domainErrors.ErrMeetingNotFound
	}
	out := *r.meeting
	out.Status = models.MeetingStatusCancelled
	r.meeting = &out
	return &out, nil
}

func (r *fakeMeetingRepo) ClaimDueMeetingReminders(ctx context.Context, within time.Duration, limit int32) ([]models.ScheduledMeeting, error) {
	return nil, nil
}

func (r *fakeMeetingRepo) MarkMeetingReminderSent(ctx context.Context, meeting *models.ScheduledMeeting) (*models.ScheduledMeeting, error) {
	if meeting == nil {
		return nil, domainErrors.ErrMeetingNotFound
	}
	out := *meeting
	now := time.Now().UTC()
	out.ReminderSentAt = &now
	r.meeting = &out
	return &out, nil
}

func (r *fakeMeetingRepo) HasActiveConsent(ctx context.Context, patientID, hospitalID uuid.UUID) (bool, error) {
	return patientID == r.ids.patientID && hospitalID == r.ids.hospitalID, nil
}

func (r *fakeMeetingRepo) GetStaffByID(ctx context.Context, staffID uuid.UUID) (*models.StaffSummary, error) {
	if r.staff == nil || r.staff.StaffID != staffID {
		return nil, domainErrors.ErrMeetingStaffNotFound
	}
	out := *r.staff
	return &out, nil
}

func (r *fakeMeetingRepo) ListSchedulableStaff(ctx context.Context, hospitalID uuid.UUID) ([]models.StaffSummary, error) {
	if r.staff == nil || r.staff.HospitalID != hospitalID {
		return nil, nil
	}
	return []models.StaffSummary{*r.staff}, nil
}

type fakeSchedulingPatientRepo struct {
	patient *models.Patient
}

func (r *fakeSchedulingPatientRepo) CreatePatient(ctx context.Context, patient *models.Patient) (*models.Patient, error) {
	return patient, nil
}
func (r *fakeSchedulingPatientRepo) GetPatientByID(ctx context.Context, id string) (*models.Patient, error) {
	if r.patient == nil || r.patient.ID.String() != id {
		return nil, domainErrors.ErrExistingPatientNotFound
	}
	return r.patient, nil
}
func (r *fakeSchedulingPatientRepo) GetPatientByUserID(ctx context.Context, userID string) (*models.Patient, error) {
	return nil, domainErrors.ErrExistingPatientNotFound
}
func (r *fakeSchedulingPatientRepo) GetPatientByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Patient, error) {
	return nil, domainErrors.ErrExistingPatientNotFound
}
func (r *fakeSchedulingPatientRepo) GetPatientByEmail(ctx context.Context, email string) (*models.Patient, error) {
	return nil, domainErrors.ErrExistingPatientNotFound
}
func (r *fakeSchedulingPatientRepo) GetPatientProfile(ctx context.Context, patientID string) (*models.PatientProfile, error) {
	return nil, domainErrors.ErrExistingPatientNotFound
}
func (r *fakeSchedulingPatientRepo) ListPatientCallSummaries(ctx context.Context, patientID string, limit, offset int32) ([]models.CallSummary, error) {
	return nil, nil
}
func (r *fakeSchedulingPatientRepo) UpdatePatient(ctx context.Context, patient *models.Patient) error {
	return nil
}
func (r *fakeSchedulingPatientRepo) DeletePatient(ctx context.Context, id string) error { return nil }

type fakeSchedulingPublisher struct {
	requested *events.MeetingRequestedData
	scheduled *events.MeetingScheduledData
	reminder  *events.MeetingReminderData
}

func (p *fakeSchedulingPublisher) PublishMeetingRequested(ctx context.Context, data *events.MeetingRequestedData) error {
	p.requested = data
	return nil
}

func (p *fakeSchedulingPublisher) PublishMeetingScheduled(ctx context.Context, data *events.MeetingScheduledData) error {
	p.scheduled = data
	return nil
}

func (p *fakeSchedulingPublisher) PublishMeetingReminder(ctx context.Context, data *events.MeetingReminderData) error {
	p.reminder = data
	return nil
}
