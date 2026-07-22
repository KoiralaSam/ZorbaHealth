package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var schedulingTracer = otel.Tracer("patient-service.scheduling")

type SchedulingService struct {
	meetings  outbound.MeetingRepository
	patients  outbound.PatientRepository
	bridges   outbound.BridgedCallRepository
	livekit   outbound.LiveKitMeetingProvider
	publisher outbound.SchedulingPublisher
	audit     auditpb.AuditServiceClient
}

func NewSchedulingService(
	meetings outbound.MeetingRepository,
	patients outbound.PatientRepository,
	bridges outbound.BridgedCallRepository,
	livekit outbound.LiveKitMeetingProvider,
	publisher outbound.SchedulingPublisher,
	audit auditpb.AuditServiceClient,
) *SchedulingService {
	return &SchedulingService{
		meetings:  meetings,
		patients:  patients,
		bridges:   bridges,
		livekit:   livekit,
		publisher: publisher,
		audit:     audit,
	}
}

func (s *SchedulingService) ScheduleHealthStaffMeeting(ctx context.Context, cmd *models.ScheduleMeetingCommand) (*models.ScheduledMeeting, error) {
	ctx, span := schedulingTracer.Start(ctx, "scheduling.schedule")
	defer span.End()

	if cmd == nil {
		return nil, domainErrors.ErrRegistrationRequestRequired
	}
	if cmd.DurationMinutes <= 0 || cmd.DurationMinutes > 480 {
		return nil, domainErrors.ErrMeetingDurationInvalid
	}
	if !cmd.StartsAt.After(time.Now().UTC()) {
		s.auditDenied(ctx, cmd, "starts_at must be in the future")
		return nil, domainErrors.ErrMeetingStartsAtInvalid
	}
	if cmd.CorrelationID == uuid.Nil {
		cmd.CorrelationID = uuid.New()
	}
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		title = "Zorba Health video visit"
	}

	staff, err := s.meetings.GetStaffByID(ctx, cmd.StaffID)
	if err != nil {
		return nil, err
	}
	if staff.HospitalID != cmd.HospitalID {
		s.auditDenied(ctx, cmd, "hospital mismatch")
		return nil, domainErrors.ErrMeetingHospitalMismatch
	}
	role := strings.ToLower(staff.Role)
	if role != "doctor" && role != "nurse" {
		return nil, domainErrors.ErrMeetingInvalidRole
	}

	ok, err := s.meetings.HasActiveConsent(ctx, cmd.PatientID, cmd.HospitalID)
	if err != nil {
		return nil, err
	}
	if !ok {
		s.auditDenied(ctx, cmd, "patient hospital consent missing")
		return nil, domainErrors.ErrMeetingConsentRequired
	}

	if err := s.requireNotificationConsent(ctx, cmd.PatientID); err != nil {
		s.auditDenied(ctx, cmd, err.Error())
		return nil, err
	}

	patient, err := s.patients.GetPatientByID(ctx, cmd.PatientID.String())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(patient.Email) == "" {
		s.auditDenied(ctx, cmd, "patient email missing")
		return nil, domainErrors.ErrMeetingPatientEmailMissing
	}

	meeting := &models.ScheduledMeeting{
		PatientID:          cmd.PatientID,
		StaffID:            cmd.StaffID,
		HospitalID:         cmd.HospitalID,
		CreatedByActorType: cmd.ActorType,
		CreatedByActorID:   cmd.ActorID,
		StartsAt:           cmd.StartsAt.UTC(),
		DurationMinutes:    cmd.DurationMinutes,
		Timezone:           cmd.Timezone,
		Title:              title,
		Notes:              strings.TrimSpace(cmd.Notes),
		Status:             models.MeetingStatusPending,
		CorrelationID:      cmd.CorrelationID,
		VoiceSessionID:     cmd.VoiceSessionID,
		SendSMS:            cmd.SendSMS,
		Channel:            cmd.Channel,
	}

	saved, err := s.meetings.Insert(ctx, meeting)
	if err != nil {
		return nil, err
	}

	span.SetAttributes(
		attribute.String("meeting.id", saved.ID.String()),
		attribute.String("correlation.id", saved.CorrelationID.String()),
	)

	notify := &events.MeetingRequestedData{
		MeetingID:       saved.ID.String(),
		PatientID:       saved.PatientID.String(),
		StaffID:         saved.StaffID.String(),
		HospitalID:      saved.HospitalID.String(),
		CorrelationID:   saved.CorrelationID.String(),
		StartsAtRFC3339: saved.StartsAt.Format(time.RFC3339),
		DurationMinutes: int(saved.DurationMinutes),
		Timezone:        saved.Timezone,
		Title:           saved.Title,
		PatientName:     patient.FullName,
		StaffEmail:      staff.Email,
		StaffName:       staff.Name,
	}
	if err := s.publisher.PublishMeetingRequested(ctx, notify); err != nil {
		span.RecordError(err)
	}

	s.auditMeeting(ctx, sharedaudit.EventMeetingRequested, saved, cmd, true, "")
	return saved, nil
}

func (s *SchedulingService) AcceptScheduledMeeting(ctx context.Context, meetingID string, actor models.ScheduleActor) (*models.ScheduledMeeting, error) {
	ctx, span := schedulingTracer.Start(ctx, "scheduling.accept")
	defer span.End()
	meeting, err := s.loadPendingForStaff(ctx, meetingID, actor)
	if err != nil {
		return nil, err
	}
	scheduled, err := s.createLiveKitRoomAndSchedule(ctx, meeting, actor, sharedaudit.EventMeetingAccepted)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return scheduled, nil
}

func (s *SchedulingService) RescheduleScheduledMeeting(ctx context.Context, meetingID string, actor models.ScheduleActor, startsAt time.Time, durationMinutes int32, timezone, title string) (*models.ScheduledMeeting, error) {
	ctx, span := schedulingTracer.Start(ctx, "scheduling.reschedule")
	defer span.End()
	meeting, err := s.loadPendingForStaff(ctx, meetingID, actor)
	if err != nil {
		return nil, err
	}
	if !startsAt.After(time.Now().UTC()) {
		return nil, domainErrors.ErrMeetingStartsAtInvalid
	}
	if durationMinutes <= 0 || durationMinutes > 480 {
		return nil, domainErrors.ErrMeetingDurationInvalid
	}
	meeting.StartsAt = startsAt.UTC()
	meeting.DurationMinutes = durationMinutes
	if strings.TrimSpace(timezone) != "" {
		meeting.Timezone = strings.TrimSpace(timezone)
	}
	if strings.TrimSpace(title) != "" {
		meeting.Title = strings.TrimSpace(title)
	}
	scheduled, err := s.createLiveKitRoomAndSchedule(ctx, meeting, actor, sharedaudit.EventMeetingRescheduled)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	return scheduled, nil
}

func (s *SchedulingService) loadPendingForStaff(ctx context.Context, meetingID string, actor models.ScheduleActor) (*models.ScheduledMeeting, error) {
	if actor.ActorType != "staff" {
		return nil, domainErrors.ErrMeetingNotFound
	}
	id, err := uuid.Parse(meetingID)
	if err != nil {
		return nil, domainErrors.ErrMeetingNotFound
	}
	meeting, err := s.meetings.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.authorizeMeetingAccess(meeting, actor); err != nil {
		return nil, err
	}
	if meeting.Status != models.MeetingStatusPending {
		return nil, domainErrors.ErrMeetingNotPending
	}
	return meeting, nil
}

func (s *SchedulingService) createLiveKitRoomAndSchedule(ctx context.Context, meeting *models.ScheduledMeeting, actor models.ScheduleActor, auditEvent string) (*models.ScheduledMeeting, error) {
	staff, err := s.meetings.GetStaffByID(ctx, meeting.StaffID)
	if err != nil {
		return nil, err
	}
	patient, err := s.patients.GetPatientByID(ctx, meeting.PatientID.String())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(patient.Email) == "" {
		return nil, domainErrors.ErrMeetingNotificationConsent
	}

	lkCtx, lkSpan := schedulingTracer.Start(ctx, "livekit.create_room")
	roomName := fmt.Sprintf("meeting-%s", meeting.ID.String())
	durationSec := meeting.DurationMinutes * 60
	if durationSec <= 0 {
		durationSec = 1800
	}
	lkResult, err := s.livekit.CreateMeetingRoom(lkCtx, outbound.LiveKitCreateInput{
		RoomName:      roomName,
		Title:         meeting.Title,
		EmptyTimeout:  uint32(durationSec + 300),
		StartsAtEpoch: meeting.StartsAt.UTC().Unix(),
		DurationSec:   durationSec,
	})
	if err != nil {
		lkSpan.RecordError(err)
		lkSpan.SetStatus(codes.Error, err.Error())
		lkSpan.End()
		s.appendAudit(ctx, sharedaudit.EventMeetingScheduleDenied, actor, meeting.PatientID.String(), map[string]any{
			"meeting_id":     meeting.ID.String(),
			"correlation_id": meeting.CorrelationID.String(),
			"denial_reason":  "livekit unavailable",
			"livekit_error":  err.Error(),
		}, false, err.Error())
		return nil, fmt.Errorf("%w: %v", domainErrors.ErrMeetingLiveKitUnavailable, err)
	}
	lkSpan.End()

	next := *meeting
	next.LiveKitRoomName = lkResult.RoomName
	next.LiveKitRoomSID = lkResult.RoomSID
	next.JoinURL = lkResult.JoinURL
	next.PatientToken = lkResult.PatientToken
	next.StaffToken = lkResult.StaffToken
	next.Status = models.MeetingStatusScheduled
	scheduled, err := s.meetings.MarkScheduled(ctx, &next)
	if err != nil {
		return nil, err
	}
	notify := &events.MeetingScheduledData{
		MeetingID:       scheduled.ID.String(),
		PatientID:       scheduled.PatientID.String(),
		StaffID:         scheduled.StaffID.String(),
		HospitalID:      scheduled.HospitalID.String(),
		CorrelationID:   scheduled.CorrelationID.String(),
		StartsAtRFC3339: scheduled.StartsAt.Format(time.RFC3339),
		DurationMinutes: int(scheduled.DurationMinutes),
		Timezone:        scheduled.Timezone,
		Title:           scheduled.Title,
		JoinURL:         scheduled.JoinURL,
		PatientEmail:    patient.Email,
		PatientPhone:    patient.PhoneNumber,
		PatientName:     patient.FullName,
		StaffEmail:      staff.Email,
		StaffName:       staff.Name,
		SendSMS:         scheduled.SendSMS,
		LiveKitRoomName: scheduled.LiveKitRoomName,
		PatientToken:    scheduled.PatientToken,
		StaffToken:      scheduled.StaffToken,
	}
	if err := s.publisher.PublishMeetingScheduled(ctx, notify); err != nil {
	}
	s.appendAudit(ctx, auditEvent, actor, scheduled.PatientID.String(), s.meetingAuditMetadata(scheduled), true, "")
	s.appendAudit(ctx, sharedaudit.EventMeetingScheduled, actor, scheduled.PatientID.String(), s.meetingAuditMetadata(scheduled), true, "")
	return scheduled, nil
}

func (s *SchedulingService) ListScheduledMeetings(ctx context.Context, filter models.ListMeetingsFilter) ([]models.ScheduledMeeting, error) {
	return s.meetings.List(ctx, filter)
}

func (s *SchedulingService) DispatchDueMeetingReminders(ctx context.Context, limit int32) (int, error) {
	ctx, span := schedulingTracer.Start(ctx, "scheduling.meeting_reminder.dispatch")
	defer span.End()
	if s.meetings == nil || s.livekit == nil || s.publisher == nil || s.patients == nil {
		return 0, domainErrors.ErrMeetingLiveKitUnavailable
	}
	meetings, err := s.meetings.ClaimDueMeetingReminders(ctx, 15*time.Minute, limit)
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, meeting := range meetings {
		staff, staffErr := s.meetings.GetStaffByID(ctx, meeting.StaffID)
		if staffErr != nil {
			span.RecordError(staffErr)
			continue
		}
		patient, patientErr := s.patients.GetPatientByID(ctx, meeting.PatientID.String())
		if patientErr != nil {
			span.RecordError(patientErr)
			continue
		}
		validFor := time.Until(meeting.StartsAt.Add(time.Duration(meeting.DurationMinutes)*time.Minute + time.Hour))
		if validFor < 2*time.Hour {
			validFor = 2 * time.Hour
		}
		patientToken, staffToken, remintErr := s.livekit.RemintMeetingTokens(ctx, meeting.LiveKitRoomName, validFor)
		if remintErr != nil {
			span.RecordError(remintErr)
			continue
		}
		meeting.PatientToken = patientToken
		meeting.StaffToken = staffToken
		updated, markErr := s.meetings.MarkMeetingReminderSent(ctx, &meeting)
		if markErr != nil {
			span.RecordError(markErr)
			continue
		}
		notify := &events.MeetingReminderData{
			MeetingID:       updated.ID.String(),
			PatientID:       updated.PatientID.String(),
			StaffID:         updated.StaffID.String(),
			HospitalID:      updated.HospitalID.String(),
			CorrelationID:   updated.CorrelationID.String(),
			StartsAtRFC3339: updated.StartsAt.Format(time.RFC3339),
			DurationMinutes: int(updated.DurationMinutes),
			Timezone:        updated.Timezone,
			Title:           updated.Title,
			JoinURL:         updated.JoinURL,
			PatientEmail:    patient.Email,
			PatientName:     patient.FullName,
			StaffEmail:      staff.Email,
			StaffName:       staff.Name,
			LiveKitRoomName: updated.LiveKitRoomName,
			PatientToken:    updated.PatientToken,
			StaffToken:      updated.StaffToken,
		}
		if pubErr := s.publisher.PublishMeetingReminder(ctx, notify); pubErr != nil {
			span.RecordError(pubErr)
			continue
		}
		sent++
	}
	span.SetAttributes(attribute.Int("meeting.reminders_sent", sent))
	return sent, nil
}

func (s *SchedulingService) CancelScheduledMeeting(ctx context.Context, meetingID string, actor models.ScheduleActor, reason string) (*models.ScheduledMeeting, error) {
	id, err := uuid.Parse(meetingID)
	if err != nil {
		return nil, domainErrors.ErrMeetingNotFound
	}
	existing, err := s.meetings.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.authorizeMeetingAccess(existing, actor); err != nil {
		return nil, err
	}
	cancelled, err := s.meetings.Cancel(ctx, id)
	if err != nil {
		return nil, err
	}
	meta := map[string]any{
		"meeting_id": cancelled.ID.String(),
		"reason":     strings.TrimSpace(reason),
	}
	s.appendAudit(ctx, sharedaudit.EventMeetingCancelled, actor, cancelled.PatientID.String(), meta, true, "")
	return cancelled, nil
}

func (s *SchedulingService) ListSchedulableStaff(ctx context.Context, patientID, hospitalID string) ([]models.StaffSummary, error) {
	pid, err := uuid.Parse(patientID)
	if err != nil {
		return nil, err
	}
	hid, err := uuid.Parse(hospitalID)
	if err != nil {
		return nil, err
	}
	ok, err := s.meetings.HasActiveConsent(ctx, pid, hid)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domainErrors.ErrMeetingConsentRequired
	}
	return s.meetings.ListSchedulableStaff(ctx, hid)
}

func (s *SchedulingService) requireNotificationConsent(ctx context.Context, patientID uuid.UUID) error {
	if s.audit == nil {
		return nil
	}
	resp, err := s.audit.CheckConsent(ctx, &auditpb.CheckConsentRequest{
		PatientId:   patientID.String(),
		ConsentType: sharedaudit.ConsentEmailNotification,
		Scope:       "",
	})
	if err != nil {
		return err
	}
	if !resp.GetAllowed() {
		return domainErrors.ErrMeetingNotificationConsent
	}
	return nil
}

func (s *SchedulingService) authorizeMeetingAccess(m *models.ScheduledMeeting, actor models.ScheduleActor) error {
	switch actor.ActorType {
	case "patient":
		if actor.PatientID != "" && actor.PatientID != m.PatientID.String() {
			return domainErrors.ErrMeetingNotFound
		}
	case "staff":
		if actor.HospitalID != "" && actor.HospitalID != m.HospitalID.String() {
			return domainErrors.ErrMeetingNotFound
		}
	}
	return nil
}

func (s *SchedulingService) auditDenied(ctx context.Context, cmd *models.ScheduleMeetingCommand, reason string) {
	meta := map[string]any{
		"patient_id":    cmd.PatientID.String(),
		"staff_id":      cmd.StaffID.String(),
		"hospital_id":   cmd.HospitalID.String(),
		"channel":       string(cmd.Channel),
		"denial_reason": reason,
	}
	actor := models.ScheduleActor{ActorType: cmd.ActorType, ActorID: cmd.ActorID}
	s.appendAudit(ctx, sharedaudit.EventMeetingScheduleDenied, actor, cmd.PatientID.String(), meta, false, reason)
}

func (s *SchedulingService) auditMeeting(ctx context.Context, eventType string, m *models.ScheduledMeeting, cmd *models.ScheduleMeetingCommand, success bool, failure string) {
	meta := s.meetingAuditMetadata(m)
	actor := models.ScheduleActor{ActorType: cmd.ActorType, ActorID: cmd.ActorID}
	s.appendAudit(ctx, eventType, actor, m.PatientID.String(), meta, success, failure)
}

func (s *SchedulingService) meetingAuditMetadata(m *models.ScheduledMeeting) map[string]any {
	meta := map[string]any{
		"meeting_id":       m.ID.String(),
		"staff_id":         m.StaffID.String(),
		"hospital_id":      m.HospitalID.String(),
		"starts_at":        m.StartsAt.Format(time.RFC3339),
		"duration_minutes": m.DurationMinutes,
		"channel":          string(m.Channel),
		"status":           string(m.Status),
		"livekit_room":     m.LiveKitRoomName,
		"correlation_id":   m.CorrelationID.String(),
	}
	return meta
}

func (s *SchedulingService) appendAudit(ctx context.Context, eventType string, actor models.ScheduleActor, patientID string, metadata map[string]any, success bool, failureReason string) {
	if s.audit == nil {
		return
	}
	metaStruct, err := structpb.NewStruct(metadata)
	if err != nil {
		metaStruct = &structpb.Struct{}
	}
	_, _ = s.audit.AppendAuditEvent(ctx, &auditpb.AppendAuditEventRequest{
		Event: &auditpb.AuditEvent{
			EventId:       uuid.NewString(),
			EventType:     eventType,
			ActorType:     actor.ActorType,
			ActorId:       actor.ActorID,
			PatientId:     patientID,
			ServiceName:   "patient-service",
			Timestamp:     timestamppb.Now(),
			SuccessStatus: success,
			FailureReason: failureReason,
			Metadata:      metaStruct,
		},
	})
}
