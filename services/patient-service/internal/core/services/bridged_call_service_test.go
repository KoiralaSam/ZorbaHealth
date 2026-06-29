package services

import (
	"context"
	"testing"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/redis/go-redis/v9"
)

type fakeBridgeRepo struct {
	sessions map[string]*models.BridgedCallSession
	lastTTL  time.Duration
}

func newFakeBridgeRepo(sessions ...*models.BridgedCallSession) *fakeBridgeRepo {
	repo := &fakeBridgeRepo{sessions: map[string]*models.BridgedCallSession{}}
	for _, session := range sessions {
		copied := *session
		repo.sessions[session.SessionID] = &copied
	}
	return repo
}

func (f *fakeBridgeRepo) Put(_ context.Context, session *models.BridgedCallSession, ttl time.Duration) error {
	copied := *session
	f.sessions[session.SessionID] = &copied
	f.lastTTL = ttl
	return nil
}

func (f *fakeBridgeRepo) Get(_ context.Context, sessionID string) (*models.BridgedCallSession, error) {
	session, ok := f.sessions[sessionID]
	if !ok {
		return nil, redis.Nil
	}
	copied := *session
	return &copied, nil
}

func (f *fakeBridgeRepo) List(_ context.Context, hospitalID string, status models.BridgedCallStatus, limit int) ([]*models.BridgedCallSession, error) {
	var out []*models.BridgedCallSession
	for _, session := range f.sessions {
		if hospitalID != "" && session.HospitalID != hospitalID {
			continue
		}
		if status != "" && session.Status != status {
			continue
		}
		copied := *session
		out = append(out, &copied)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func bridgeTestSession() *models.BridgedCallSession {
	return &models.BridgedCallSession{
		SessionID:          "room-1",
		RoomSID:            "room-1",
		PatientID:          "patient-1",
		HospitalID:         "hospital-1",
		Status:             models.BridgedCallStatusTransferRequested,
		PatientAccessToken: "patient-jwt",
		RequestedAt:        time.Now().UTC(),
	}
}

func bridgeTestService(repo *fakeBridgeRepo) *SchedulingService {
	return NewSchedulingService(nil, nil, repo, &fakeLiveKitProvider{}, &fakeSchedulingPublisher{}, nil)
}

func staffActor(staffID string) models.ScheduleActor {
	return models.ScheduleActor{
		ActorType:  "staff",
		ActorID:    staffID,
		StaffID:    staffID,
		HospitalID: "hospital-1",
	}
}

func TestConnectBridgedCallStoresStaffTokenAndMintsJoinToken(t *testing.T) {
	repo := newFakeBridgeRepo(bridgeTestSession())
	svc := bridgeTestService(repo)

	result, err := svc.ConnectBridgedCall(context.Background(), "room-1", staffActor("staff-1"), "nurse-jane", "staff-jwt")
	if err != nil {
		t.Fatalf("ConnectBridgedCall: %v", err)
	}
	if result.StaffRoomToken == "" || result.LiveKitWSURL == "" {
		t.Fatalf("expected LiveKit join credentials, got %+v", result)
	}
	stored := repo.sessions["room-1"]
	if stored.StaffAccessToken != "staff-jwt" {
		t.Fatalf("staff token = %q, want staff-jwt", stored.StaffAccessToken)
	}
	if stored.PatientAccessToken != "patient-jwt" {
		t.Fatalf("patient token = %q, want unchanged patient-jwt", stored.PatientAccessToken)
	}
	if stored.Status != models.BridgedCallStatusConnected {
		t.Fatalf("status = %q, want connected", stored.Status)
	}
}

func TestUpdateBridgedCallTranslationRefreshesPatientToken(t *testing.T) {
	repo := newFakeBridgeRepo(bridgeTestSession())
	svc := bridgeTestService(repo)

	_, err := svc.UpdateBridgedCallTranslation(context.Background(), &models.UpdateBridgedCallTranslationCommand{
		SessionID:   "room-1",
		Participant: models.BridgedCallParticipantPatient,
		Preferences: models.BridgedCallTranslationPreferences{Enabled: true, LanguageCode: "ES"},
		ActorType:   "patient",
		ActorID:     "patient-1",
		PatientID:   "patient-1",
		AccessToken: "patient-jwt-rotated",
	})
	if err != nil {
		t.Fatalf("UpdateBridgedCallTranslation: %v", err)
	}
	stored := repo.sessions["room-1"]
	if stored.PatientAccessToken != "patient-jwt-rotated" {
		t.Fatalf("patient token = %q, want refreshed token", stored.PatientAccessToken)
	}
	if stored.PatientTranslation.LanguageCode != "es" {
		t.Fatalf("language code = %q, want normalized es", stored.PatientTranslation.LanguageCode)
	}
}

func TestConnectBridgedCallRejectsDifferentStaffWhenAssigned(t *testing.T) {
	session := bridgeTestSession()
	session.StaffID = "staff-1"
	repo := newFakeBridgeRepo(session)
	svc := bridgeTestService(repo)

	_, err := svc.ConnectBridgedCall(context.Background(), "room-1", staffActor("staff-2"), "", "other-jwt")
	if err != domainErrors.ErrBridgedCallForbidden {
		t.Fatalf("err = %v, want ErrBridgedCallForbidden", err)
	}
}

func TestEndBridgedCallClearsTokensAndShortensTTL(t *testing.T) {
	session := bridgeTestSession()
	session.Status = models.BridgedCallStatusConnected
	session.StaffAccessToken = "staff-jwt"
	repo := newFakeBridgeRepo(session)
	svc := bridgeTestService(repo)

	ended, err := svc.EndBridgedCall(context.Background(), "room-1", models.ScheduleActor{
		ActorType: "patient",
		ActorID:   "patient-1",
		PatientID: "patient-1",
	}, "patient hung up")
	if err != nil {
		t.Fatalf("EndBridgedCall: %v", err)
	}
	if ended.Status != models.BridgedCallStatusEnded {
		t.Fatalf("status = %q, want ended", ended.Status)
	}
	stored := repo.sessions["room-1"]
	if stored.PatientAccessToken != "" || stored.StaffAccessToken != "" {
		t.Fatalf("tokens not cleared: %+v", stored)
	}
	if repo.lastTTL != bridgedCallEndedTTL {
		t.Fatalf("ttl = %v, want %v", repo.lastTTL, bridgedCallEndedTTL)
	}
}

func TestListBridgedCallSessionsRequiresStaffHospital(t *testing.T) {
	repo := newFakeBridgeRepo(bridgeTestSession())
	svc := bridgeTestService(repo)

	if _, err := svc.ListBridgedCallSessions(context.Background(), models.ScheduleActor{
		ActorType: "patient",
		PatientID: "patient-1",
	}, "", 10); err != domainErrors.ErrBridgedCallForbidden {
		t.Fatalf("err = %v, want ErrBridgedCallForbidden", err)
	}

	sessions, err := svc.ListBridgedCallSessions(context.Background(), staffActor("staff-1"), "transfer_requested", 10)
	if err != nil {
		t.Fatalf("ListBridgedCallSessions: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != "room-1" {
		t.Fatalf("sessions = %+v, want the pending transfer", sessions)
	}
}
