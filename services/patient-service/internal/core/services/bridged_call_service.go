package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
)

const (
	bridgedCallSessionTTL = 12 * time.Hour
	// Ended sessions stay around briefly for audit/UX, then expire quickly so
	// stored JWTs do not outlive the consult.
	bridgedCallEndedTTL = 15 * time.Minute
)

func (s *SchedulingService) RequestBridgedCallTransfer(ctx context.Context, cmd *models.RequestBridgedCallTransferCommand) (*models.BridgedCallSession, error) {
	ctx, span := schedulingTracer.Start(ctx, "bridge.transfer.request")
	defer span.End()

	if cmd == nil || strings.TrimSpace(cmd.SessionID) == "" {
		return nil, domainErrors.ErrBridgedCallSessionRequired
	}
	if strings.TrimSpace(cmd.HospitalID) == "" {
		return nil, domainErrors.ErrBridgedCallHospitalRequired
	}
	if s.bridges == nil {
		return nil, domainErrors.ErrBridgedCallStoreUnavailable
	}
	if strings.TrimSpace(cmd.PatientID) == "" {
		return nil, domainErrors.ErrMeetingNotFound
	}

	patientUUID, err := uuid.Parse(strings.TrimSpace(cmd.PatientID))
	if err != nil {
		return nil, domainErrors.ErrMeetingNotFound
	}
	hospitalUUID, err := uuid.Parse(strings.TrimSpace(cmd.HospitalID))
	if err != nil {
		return nil, domainErrors.ErrMeetingHospitalMismatch
	}
	consented, err := s.meetings.HasActiveConsent(ctx, patientUUID, hospitalUUID)
	if err != nil {
		return nil, err
	}
	if !consented {
		return nil, domainErrors.ErrMeetingConsentRequired
	}

	session := &models.BridgedCallSession{
		SessionID:            strings.TrimSpace(cmd.SessionID),
		RoomSID:              fallbackBridgeRoom(strings.TrimSpace(cmd.RoomSID), strings.TrimSpace(cmd.SessionID)),
		PatientID:            strings.TrimSpace(cmd.PatientID),
		HospitalID:           strings.TrimSpace(cmd.HospitalID),
		StaffID:              strings.TrimSpace(cmd.StaffID),
		Status:               models.BridgedCallStatusTransferRequested,
		RequestedByActorType: cmd.ActorType,
		RequestedByActorID:   cmd.ActorID,
		PatientAccessToken:   strings.TrimSpace(cmd.AccessToken),
		TransferReason:       strings.TrimSpace(cmd.Reason),
		RequestedAt:          time.Now().UTC(),
		PatientTranslation: models.BridgedCallTranslationPreferences{
			LanguageMode: models.TranslationModeAuto,
		},
		StaffTranslation: models.BridgedCallTranslationPreferences{
			LanguageMode: models.TranslationModeAuto,
		},
	}
	if err := s.bridges.Put(ctx, session, bridgedCallSessionTTL); err != nil {
		return nil, err
	}
	span.SetAttributes(
		attribute.String("voice.session_id", session.SessionID),
		attribute.String("hospital.id", session.HospitalID),
	)
	s.appendAudit(ctx, sharedaudit.EventCallTransferRequested, models.ScheduleActor{
		ActorType:  cmd.ActorType,
		ActorID:    cmd.ActorID,
		PatientID:  cmd.PatientID,
		HospitalID: cmd.HospitalID,
	}, cmd.PatientID, map[string]any{
		"session_id":  session.SessionID,
		"room_sid":    session.RoomSID,
		"hospital_id": session.HospitalID,
		"staff_id":    session.StaffID,
		"status":      string(session.Status),
	}, true, "")
	return session, nil
}

func (s *SchedulingService) ConnectBridgedCall(ctx context.Context, sessionID string, actor models.ScheduleActor, participantIdentity string, joinMode string, accessToken string) (*models.BridgedCallConnectResult, error) {
	ctx, span := schedulingTracer.Start(ctx, "bridge.transfer.connect")
	defer span.End()

	joinMode = strings.ToLower(strings.TrimSpace(joinMode))
	if joinMode == "" {
		joinMode = "web"
	}
	if joinMode != "web" && joinMode != "phone" {
		return nil, domainErrors.ErrBridgedCallInvalidJoinMode
	}

	session, err := s.GetBridgedCallSession(ctx, sessionID, actor)
	if err != nil {
		return nil, err
	}
	if session.Status == models.BridgedCallStatusEnded {
		return nil, domainErrors.ErrBridgedCallAlreadyEnded
	}
	now := time.Now().UTC()
	// Snapshot whether each party already configured translation before we
	// stamp any of their fields, so connect-time defaults never clobber an
	// explicit choice made via the preference-update UIs.
	staffConfigured := !session.StaffTranslation.UpdatedAt.IsZero()
	patientConfigured := !session.PatientTranslation.UpdatedAt.IsZero()

	session.Status = models.BridgedCallStatusConnected
	session.ConnectedAt = &now
	if actor.StaffID != "" {
		session.StaffID = actor.StaffID
	}
	refreshBridgeActorToken(session, actor, accessToken)

	result := &models.BridgedCallConnectResult{Session: session}
	roomName := s.bridgeRoomName(ctx, session)

	if joinMode == "phone" {
		if actor.StaffID == "" || s.meetings == nil || s.livekit == nil {
			return nil, domainErrors.ErrBridgedCallStaffPhoneRequired
		}
		staffUUID, parseErr := uuid.Parse(actor.StaffID)
		if parseErr != nil {
			return nil, domainErrors.ErrMeetingStaffNotFound
		}
		staff, staffErr := s.meetings.GetStaffByID(ctx, staffUUID)
		if staffErr != nil {
			return nil, staffErr
		}
		phone := strings.TrimSpace(staff.PhoneNumber)
		if phone == "" {
			return nil, domainErrors.ErrBridgedCallStaffPhoneRequired
		}
		staffIdentity := normalizeStaffIdentity("staff-sip-"+actor.StaffID, session.SessionID)
		dial, dialErr := s.livekit.DialSIPParticipant(ctx, outbound.DialSIPParticipantInput{
			RoomName:            roomName,
			PhoneNumber:         phone,
			ParticipantIdentity: staffIdentity,
			ParticipantName:     staff.Name,
		})
		if dialErr != nil {
			span.RecordError(dialErr)
			return nil, dialErr
		}
		if dial != nil && strings.TrimSpace(dial.ParticipantIdentity) != "" {
			staffIdentity = dial.ParticipantIdentity
		}
		session.StaffTranslation.ParticipantIdentity = staffIdentity
		session.StaffTranslation.UpdatedAt = now
	} else {
		// The voice agent only enters interpreter mode for participants whose
		// identity starts with "staff-" (is_staff_identity). Normalize here so a
		// clinician joins as staff regardless of what the hospital console typed.
		staffIdentity := normalizeStaffIdentity(participantIdentity, session.SessionID)
		if participantIdentity != "" || actor.ActorType == sharedauth.ActorStaff {
			session.StaffTranslation.ParticipantIdentity = staffIdentity
			session.StaffTranslation.UpdatedAt = now
		}
		if actor.ActorType == sharedauth.ActorStaff && s.livekit != nil {
			roomToken, lkErr := s.livekit.MintRoomJoinToken(ctx, roomName, staffIdentity)
			if lkErr != nil {
				// Connect still succeeds as control-plane state; staff can retry
				// the realtime join, and the relay path is unaffected.
				span.RecordError(lkErr)
			} else {
				result.StaffRoomToken = roomToken.Token
				result.LiveKitWSURL = roomToken.WSURL
				if session.StaffTranslation.ParticipantIdentity == "" {
					session.StaffTranslation.ParticipantIdentity = staffIdentity
					session.StaffTranslation.UpdatedAt = now
				}
			}
		}
	}

	// Arm interpretation in both directions when the clinician joins so the
	// bridge interprets immediately. Defaults only apply to a party that has
	// not already configured its own preferences via the update APIs.
	enableBridgeTranslationDefaults(session, now, staffConfigured, patientConfigured)

	if err := s.bridges.Put(ctx, session, bridgedCallSessionTTL); err != nil {
		return nil, err
	}
	span.SetAttributes(
		attribute.String("voice.session_id", session.SessionID),
		attribute.String("bridge.join_mode", joinMode),
	)
	s.appendAudit(ctx, sharedaudit.EventCallTransferConnected, actor, session.PatientID, map[string]any{
		"session_id":                 session.SessionID,
		"room_sid":                   session.RoomSID,
		"hospital_id":                session.HospitalID,
		"staff_id":                   session.StaffID,
		"join_mode":                  joinMode,
		"staff_participant_identity": session.StaffTranslation.ParticipantIdentity,
	}, true, "")
	s.appendAudit(ctx, sharedaudit.EventInterpretationSessionStarted, actor, session.PatientID, map[string]any{
		"session_id":  session.SessionID,
		"room_sid":    session.RoomSID,
		"hospital_id": session.HospitalID,
		"staff_id":    session.StaffID,
		"join_mode":   joinMode,
	}, true, "")
	return result, nil
}

func (s *SchedulingService) GetBridgedCallSession(ctx context.Context, sessionID string, actor models.ScheduleActor) (*models.BridgedCallSession, error) {
	if s.bridges == nil {
		return nil, domainErrors.ErrBridgedCallStoreUnavailable
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, domainErrors.ErrBridgedCallSessionRequired
	}
	session, err := s.bridges.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domainErrors.ErrBridgedCallNotFound
		}
		return nil, err
	}
	if !bridgeActorAllowed(session, actor) {
		return nil, domainErrors.ErrBridgedCallForbidden
	}
	return session, nil
}

func (s *SchedulingService) MintBridgedCallPatientToken(ctx context.Context, session *models.BridgedCallSession) (*outbound.LiveKitRoomToken, error) {
	if session == nil || s.livekit == nil {
		return nil, nil
	}
	roomName := s.bridgeRoomName(ctx, session)
	if roomName == "" {
		return nil, domainErrors.ErrBridgedCallSessionRequired
	}
	identity := fmt.Sprintf("patient-app-%d", time.Now().UnixNano())
	return s.livekit.MintRoomJoinToken(ctx, roomName, identity)
}

func (s *SchedulingService) UpdateBridgedCallTranslation(ctx context.Context, cmd *models.UpdateBridgedCallTranslationCommand) (*models.BridgedCallSession, error) {
	ctx, span := schedulingTracer.Start(ctx, "bridge.translation.update")
	defer span.End()

	if cmd == nil || strings.TrimSpace(cmd.SessionID) == "" {
		return nil, domainErrors.ErrBridgedCallSessionRequired
	}
	if cmd.Participant != models.BridgedCallParticipantPatient && cmd.Participant != models.BridgedCallParticipantStaff {
		return nil, domainErrors.ErrBridgedCallInvalidParticipant
	}
	mode := cmd.Preferences.LanguageMode
	if mode == "" {
		mode = models.TranslationModeAuto
	}
	if mode != models.TranslationModeAuto && mode != models.TranslationModeManual {
		return nil, domainErrors.ErrBridgedCallInvalidMode
	}

	session, err := s.GetBridgedCallSession(ctx, cmd.SessionID, models.ScheduleActor{
		ActorType:  cmd.ActorType,
		ActorID:    cmd.ActorID,
		StaffID:    cmd.StaffID,
		PatientID:  cmd.PatientID,
		HospitalID: cmd.HospitalID,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	prefs := cmd.Preferences
	prefs.LanguageMode = mode
	prefs.LanguageCode = strings.TrimSpace(strings.ToLower(prefs.LanguageCode))
	prefs.ParticipantIdentity = strings.TrimSpace(prefs.ParticipantIdentity)
	prefs.UpdatedAt = now

	// Keep the relay's forwarded JWT fresh: every authenticated preference
	// update re-stamps the caller's token so long consults outlive the
	// originally captured JWT.
	refreshBridgeActorToken(session, models.ScheduleActor{
		ActorType: cmd.ActorType,
	}, cmd.AccessToken)

	switch cmd.Participant {
	case models.BridgedCallParticipantPatient:
		session.PatientTranslation = prefs
	case models.BridgedCallParticipantStaff:
		session.StaffTranslation = prefs
	}
	if err := s.bridges.Put(ctx, session, bridgedCallSessionTTL); err != nil {
		return nil, err
	}
	span.SetAttributes(
		attribute.String("voice.session_id", session.SessionID),
		attribute.String("bridge.participant", string(cmd.Participant)),
		attribute.String("bridge.language_mode", string(prefs.LanguageMode)),
	)
	s.appendAudit(ctx, sharedaudit.EventInterpretationPreferencesUpdated, models.ScheduleActor{
		ActorType:  cmd.ActorType,
		ActorID:    cmd.ActorID,
		PatientID:  session.PatientID,
		HospitalID: session.HospitalID,
	}, session.PatientID, map[string]any{
		"session_id":           session.SessionID,
		"participant":          string(cmd.Participant),
		"enabled":              prefs.Enabled,
		"language_mode":        string(prefs.LanguageMode),
		"language_code":        prefs.LanguageCode,
		"participant_identity": prefs.ParticipantIdentity,
	}, true, "")
	return session, nil
}

func (s *SchedulingService) EndBridgedCall(ctx context.Context, sessionID string, actor models.ScheduleActor, reason string) (*models.BridgedCallSession, error) {
	ctx, span := schedulingTracer.Start(ctx, "bridge.transfer.end")
	defer span.End()

	session, err := s.GetBridgedCallSession(ctx, sessionID, actor)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	session.Status = models.BridgedCallStatusEnded
	session.EndedAt = &now
	if strings.TrimSpace(reason) != "" {
		session.TransferReason = strings.TrimSpace(reason)
	}
	// Drop stored JWTs immediately on end; the relay must not translate for a
	// finished consult.
	session.PatientAccessToken = ""
	session.StaffAccessToken = ""
	if err := s.bridges.Put(ctx, session, bridgedCallEndedTTL); err != nil {
		return nil, err
	}
	span.SetAttributes(attribute.String("voice.session_id", session.SessionID))
	s.appendAudit(ctx, sharedaudit.EventCallBridgedEnded, actor, session.PatientID, map[string]any{
		"session_id":  session.SessionID,
		"room_sid":    session.RoomSID,
		"hospital_id": session.HospitalID,
		"staff_id":    session.StaffID,
		"reason":      strings.TrimSpace(reason),
	}, true, "")
	s.appendAudit(ctx, sharedaudit.EventInterpretationSessionEnded, actor, session.PatientID, map[string]any{
		"session_id":  session.SessionID,
		"room_sid":    session.RoomSID,
		"hospital_id": session.HospitalID,
		"staff_id":    session.StaffID,
		"reason":      strings.TrimSpace(reason),
	}, true, "")
	return session, nil
}

// ListBridgedCallSessions returns bridged sessions for the staff actor's
// hospital, optionally filtered by status (e.g. transfer_requested).
func (s *SchedulingService) ListBridgedCallSessions(ctx context.Context, actor models.ScheduleActor, status string, limit int) ([]*models.BridgedCallSession, error) {
	ctx, span := schedulingTracer.Start(ctx, "bridge.sessions.list")
	defer span.End()

	if s.bridges == nil {
		return nil, domainErrors.ErrBridgedCallStoreUnavailable
	}
	if actor.ActorType != sharedauth.ActorStaff || strings.TrimSpace(actor.HospitalID) == "" {
		return nil, domainErrors.ErrBridgedCallForbidden
	}
	sessions, err := s.bridges.List(ctx, actor.HospitalID, models.BridgedCallStatus(strings.TrimSpace(status)), limit)
	if err != nil {
		return nil, err
	}
	span.SetAttributes(
		attribute.String("hospital.id", actor.HospitalID),
		attribute.Int("bridge.session_count", len(sessions)),
	)
	return sessions, nil
}

// refreshBridgeActorToken re-stamps the stored JWT for whichever party the
// authenticated actor represents. Empty tokens never overwrite stored ones.
func refreshBridgeActorToken(session *models.BridgedCallSession, actor models.ScheduleActor, accessToken string) {
	accessToken = strings.TrimSpace(accessToken)
	if session == nil || accessToken == "" {
		return
	}
	switch actor.ActorType {
	case sharedauth.ActorPatient:
		session.PatientAccessToken = accessToken
	case sharedauth.ActorStaff:
		session.StaffAccessToken = accessToken
	}
}

func bridgeActorAllowed(session *models.BridgedCallSession, actor models.ScheduleActor) bool {
	if session == nil {
		return false
	}
	switch actor.ActorType {
	case sharedauth.ActorPatient:
		return actor.PatientID != "" && actor.PatientID == session.PatientID
	case sharedauth.ActorStaff:
		if actor.HospitalID == "" || actor.HospitalID != session.HospitalID {
			return false
		}
		// When the patient named a specific staff member (or one already
		// connected), only that staff member may act on the session.
		if session.StaffID != "" {
			return actor.StaffID == session.StaffID
		}
		return true
	default:
		return false
	}
}

// normalizeStaffIdentity guarantees the clinician's LiveKit identity carries
// the "staff-" prefix the voice agent keys interpreter mode on. An empty
// identity falls back to a deterministic per-session value.
func normalizeStaffIdentity(participantIdentity, sessionID string) string {
	id := strings.TrimSpace(participantIdentity)
	if id == "" {
		return "staff-" + strings.TrimSpace(sessionID)
	}
	if !strings.HasPrefix(strings.ToLower(id), "staff-") {
		return "staff-" + id
	}
	return id
}

func fallbackBridgeRoom(roomSID, sessionID string) string {
	if roomSID != "" {
		return roomSID
	}
	return sessionID
}

// bridgeRoomName resolves the LiveKit room NAME a join token must target.
// Bridged sessions store the voice session SID ("RM_..."), but join-token
// grants are keyed by room name; minting against the SID drops the joiner into
// a brand-new empty room instead of the live SIP call. Resolution failures fall
// back to the stored value so behaviour never regresses below the old code.
func (s *SchedulingService) bridgeRoomName(ctx context.Context, session *models.BridgedCallSession) string {
	name := fallbackBridgeRoom(strings.TrimSpace(session.RoomSID), strings.TrimSpace(session.SessionID))
	if s.livekit == nil || name == "" {
		return name
	}
	resolved, err := s.livekit.ResolveRoomName(ctx, name)
	if err != nil {
		return name
	}
	if resolved = strings.TrimSpace(resolved); resolved != "" {
		return resolved
	}
	return name
}

// enableBridgeTranslationDefaults arms interpretation for both parties when the
// clinician connects. It is non-destructive: a party that already configured
// its own preferences (non-zero UpdatedAt) keeps them untouched. The clinician
// defaults to reading English; the patient direction is enabled in auto mode so
// the patient's spoken language drives source detection, with the target
// LanguageCode left to the preference-update flow / SIP-detected language.
func enableBridgeTranslationDefaults(session *models.BridgedCallSession, now time.Time, staffConfigured, patientConfigured bool) {
	if !staffConfigured {
		session.StaffTranslation.Enabled = true
		if session.StaffTranslation.LanguageMode == "" {
			session.StaffTranslation.LanguageMode = models.TranslationModeAuto
		}
		if strings.TrimSpace(session.StaffTranslation.LanguageCode) == "" {
			session.StaffTranslation.LanguageCode = "en"
		}
		session.StaffTranslation.UpdatedAt = now
	}
	if !patientConfigured {
		session.PatientTranslation.Enabled = true
		if session.PatientTranslation.LanguageMode == "" {
			session.PatientTranslation.LanguageMode = models.TranslationModeAuto
		}
		session.PatientTranslation.UpdatedAt = now
	}
}
