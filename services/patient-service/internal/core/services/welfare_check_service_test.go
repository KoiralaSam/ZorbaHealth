package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type fakePatientRepo struct {
	outbound.PatientRepository
	patient *models.Patient
}

func (f *fakePatientRepo) GetPatientByID(ctx context.Context, id string) (*models.Patient, error) {
	if f.patient == nil {
		return nil, errors.New("not found")
	}
	return f.patient, nil
}

type fakeWelfareRepo struct {
	inserted     *models.WelfareCheck
	cancelled    *models.WelfareCheck
	claimed      []models.WelfareCheckRun
	dispatched   []models.WelfareCheckDispatchResult
	failed       []fakeWelfareFail
	missed       []uuid.UUID
	lifecycle    *models.WelfareCheckRun
	lifecycleErr error
	failMarkErr  error
	cancelErr    error
	claimErr     error
	insertErr    error
}

type fakeWelfareFail struct {
	RunID         uuid.UUID
	Reason        string
	NextAttemptAt *time.Time
}

func (f *fakeWelfareRepo) InsertWelfareCheck(ctx context.Context, check *models.WelfareCheck) (*models.WelfareCheck, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	out := *check
	out.ID = uuid.New()
	out.CreatedAt = time.Now().UTC()
	out.UpdatedAt = out.CreatedAt
	f.inserted = &out
	return &out, nil
}

func (f *fakeWelfareRepo) ListWelfareChecks(ctx context.Context, filter models.ListWelfareChecksFilter) ([]models.WelfareCheck, error) {
	return nil, nil
}

func (f *fakeWelfareRepo) CancelWelfareCheck(ctx context.Context, patientID, checkID uuid.UUID) (*models.WelfareCheck, error) {
	if f.cancelErr != nil {
		return nil, f.cancelErr
	}
	return f.cancelled, nil
}

func (f *fakeWelfareRepo) ClaimDueWelfareCheckRuns(ctx context.Context, limit int32) ([]models.WelfareCheckRun, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	return f.claimed, nil
}

func (f *fakeWelfareRepo) PersistWelfareRunLiveKitResult(ctx context.Context, result models.WelfareCheckDispatchResult) error {
	return nil
}

func (f *fakeWelfareRepo) MarkWelfareRunDispatched(ctx context.Context, result models.WelfareCheckDispatchResult) error {
	f.dispatched = append(f.dispatched, result)
	return nil
}

func (f *fakeWelfareRepo) MarkWelfareRunFailed(ctx context.Context, runID uuid.UUID, reason string, nextAttemptAt *time.Time) error {
	if f.failMarkErr != nil {
		return f.failMarkErr
	}
	f.failed = append(f.failed, fakeWelfareFail{RunID: runID, Reason: reason, NextAttemptAt: nextAttemptAt})
	return nil
}

func (f *fakeWelfareRepo) MarkWelfareRunMissed(ctx context.Context, runID uuid.UUID, reason string) error {
	f.missed = append(f.missed, runID)
	return nil
}

func (f *fakeWelfareRepo) UpdateWelfareRunLifecycle(ctx context.Context, patientID, runID uuid.UUID, status models.WelfareCheckRunStatus, reason string) (*models.WelfareCheckRun, error) {
	if f.lifecycleErr != nil {
		return nil, f.lifecycleErr
	}
	if f.lifecycle == nil {
		return nil, domainErrors.ErrWelfareCheckRunNotFound
	}
	out := *f.lifecycle
	out.Status = status
	out.FailureReason = reason
	return &out, nil
}

func (f *fakeWelfareRepo) GetWelfareCheckRun(ctx context.Context, patientID, runID uuid.UUID) (*models.WelfareCheckRun, error) {
	return f.lifecycle, nil
}

type fakeAuthRepo struct {
	outbound.AuthRepository
}

func (f *fakeAuthRepo) CreatePatientSession(ctx context.Context, userID string, scopes []string) (*models.LoginResult, error) {
	return &models.LoginResult{AccessToken: "token-" + userID}, nil
}

type fakeCallProvider struct {
	calls []outbound.WelfareCheckCallInput
	err   error
}

func (f *fakeCallProvider) StartWelfareCheckCall(ctx context.Context, in outbound.WelfareCheckCallInput) (*outbound.WelfareCheckCallResult, error) {
	f.calls = append(f.calls, in)
	if f.err != nil {
		return nil, f.err
	}
	return &outbound.WelfareCheckCallResult{
		RoomName:   in.RoomName,
		RoomSID:    "sid",
		DispatchID: "dispatch",
		SIPCallID:  "sip",
	}, nil
}

type fakeAuditClient struct {
	auditpb.AuditServiceClient
	allowed bool
	err     error
}

func (f *fakeAuditClient) CheckConsent(ctx context.Context, in *auditpb.CheckConsentRequest, opts ...grpc.CallOption) (*auditpb.CheckConsentResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &auditpb.CheckConsentResponse{Allowed: f.allowed}, nil
}

func (f *fakeAuditClient) AppendAuditEvent(ctx context.Context, in *auditpb.AppendAuditEventRequest, opts ...grpc.CallOption) (*auditpb.AppendAuditEventResponse, error) {
	return &auditpb.AppendAuditEventResponse{}, nil
}

func newTestService(patient *models.Patient, welfare *fakeWelfareRepo, audit auditpb.AuditServiceClient, calls outbound.WelfareCheckCallProvider) *PatientService {
	return &PatientService{
		repo:          &fakePatientRepo{patient: patient},
		welfareChecks: welfare,
		authService:   &fakeAuthRepo{},
		audit:         audit,
		welfareCalls:  calls,
	}
}

func TestCreateWelfareCheckValidatesTimezoneAndConsent(t *testing.T) {
	t.Setenv("WELFARE_CHECK_ALLOW_CONSENT_BYPASS", "false")
	patientID := uuid.New()
	patient := &models.Patient{
		ID:          patientID,
		UserID:      uuid.New(),
		PhoneNumber: "+15551234567",
		FullName:    "Test Patient",
	}
	welfare := &fakeWelfareRepo{}
	svc := newTestService(patient, welfare, &fakeAuditClient{allowed: true}, &fakeCallProvider{})

	_, err := svc.CreateWelfareCheck(context.Background(), &models.CreateWelfareCheckCommand{
		PatientID:   patientID,
		ScheduledAt: time.Now().UTC().Add(10 * time.Minute),
		Timezone:    "Not/AZone",
		ReasonCode:  models.WelfareReasonDailyCheckup,
	})
	if !errors.Is(err, domainErrors.ErrWelfareCheckTimezoneInvalid) {
		t.Fatalf("expected timezone error, got %v", err)
	}

	svcNoAudit := newTestService(patient, welfare, nil, &fakeCallProvider{})
	_, err = svcNoAudit.CreateWelfareCheck(context.Background(), &models.CreateWelfareCheckCommand{
		PatientID:   patientID,
		ScheduledAt: time.Now().UTC().Add(10 * time.Minute),
		Timezone:    "America/Chicago",
		ReasonCode:  models.WelfareReasonDailyCheckup,
	})
	if !errors.Is(err, domainErrors.ErrWelfareCheckConsentUnavailable) {
		t.Fatalf("expected consent unavailable, got %v", err)
	}

	svcDenied := newTestService(patient, welfare, &fakeAuditClient{allowed: false}, &fakeCallProvider{})
	_, err = svcDenied.CreateWelfareCheck(context.Background(), &models.CreateWelfareCheckCommand{
		PatientID:   patientID,
		ScheduledAt: time.Now().UTC().Add(10 * time.Minute),
		Timezone:    "UTC",
		ReasonCode:  models.WelfareReasonDailyCheckup,
	})
	if !errors.Is(err, domainErrors.ErrWelfareCheckConsentRequired) {
		t.Fatalf("expected consent required, got %v", err)
	}

	_, err = svc.CreateWelfareCheck(context.Background(), &models.CreateWelfareCheckCommand{
		PatientID:    patientID,
		ScheduledAt:  time.Now().UTC().Add(15 * time.Minute),
		Timezone:     "America/Chicago",
		ReasonCode:   models.WelfareReasonMentalWellbeing,
		ReasonDetail: "feeling low",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if welfare.inserted == nil || welfare.inserted.ReasonCode != models.WelfareReasonMentalWellbeing {
		t.Fatalf("expected inserted welfare check")
	}
}

func TestDispatchUsesStableRoomAndPersistsTokenOnlyInCallInput(t *testing.T) {
	t.Setenv("WELFARE_CHECK_ALLOW_CONSENT_BYPASS", "true")
	runID := uuid.New()
	requestID := uuid.New()
	patientID := uuid.New()
	welfare := &fakeWelfareRepo{
		claimed: []models.WelfareCheckRun{{
			ID:                 runID,
			RequestID:          requestID,
			PatientID:          patientID,
			ScheduledAt:        time.Now().UTC().Add(-time.Minute),
			Status:             models.WelfareRunStatusClaimed,
			Attempts:           1,
			LiveKitRoomName:    "welfare-check-" + runID.String(),
			RequestReasonCode:  models.WelfareReasonDailyCheckup,
			RequestTimezone:    "UTC",
			PatientPhoneNumber: "+15551234567",
			PatientFullName:    "Test",
			PatientUserID:      uuid.New(),
		}},
	}
	calls := &fakeCallProvider{}
	svc := newTestService(&models.Patient{ID: patientID, PhoneNumber: "+15551234567"}, welfare, nil, calls)
	results, err := svc.DispatchDueWelfareChecks(context.Background(), 10)
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(calls.calls) != 1 {
		t.Fatalf("expected one outbound call")
	}
	if calls.calls[0].RoomName != "welfare-check-"+runID.String() {
		t.Fatalf("unexpected room %s", calls.calls[0].RoomName)
	}
	if calls.calls[0].PatientToken == "" {
		t.Fatalf("expected patient token in agent-private call input")
	}
	if len(welfare.dispatched) != 1 {
		t.Fatalf("expected dispatched mark")
	}
}

func TestUpdateWelfareRunLifecycleRejectsIllegalStatus(t *testing.T) {
	welfare := &fakeWelfareRepo{
		lifecycle: &models.WelfareCheckRun{
			ID:        uuid.New(),
			RequestID: uuid.New(),
			PatientID: uuid.New(),
			Status:    models.WelfareRunStatusDispatched,
		},
	}
	svc := newTestService(nil, welfare, &fakeAuditClient{allowed: true}, &fakeCallProvider{})
	_, err := svc.UpdateWelfareRunLifecycle(context.Background(), &models.UpdateWelfareRunLifecycleCommand{
		PatientID: welfare.lifecycle.PatientID.String(),
		RunID:     welfare.lifecycle.ID.String(),
		Status:    models.WelfareRunStatusPending,
	})
	if !errors.Is(err, domainErrors.ErrWelfareCheckRunTransition) {
		t.Fatalf("expected transition error, got %v", err)
	}

	run, err := svc.UpdateWelfareRunLifecycle(context.Background(), &models.UpdateWelfareRunLifecycleCommand{
		PatientID: welfare.lifecycle.PatientID.String(),
		RunID:     welfare.lifecycle.ID.String(),
		Status:    models.WelfareRunStatusAnswered,
	})
	if err != nil {
		t.Fatalf("answered update failed: %v", err)
	}
	if run.Status != models.WelfareRunStatusAnswered {
		t.Fatalf("unexpected status %s", run.Status)
	}
}

func TestDispatchAllFailuresSurfaceError(t *testing.T) {
	t.Setenv("WELFARE_CHECK_ALLOW_CONSENT_BYPASS", "true")
	welfare := &fakeWelfareRepo{
		claimed: []models.WelfareCheckRun{{
			ID:                 uuid.New(),
			RequestID:          uuid.New(),
			PatientID:          uuid.New(),
			Attempts:           1,
			PatientPhoneNumber: "+15551234567",
			PatientUserID:      uuid.New(),
			RequestReasonCode:  models.WelfareReasonOther,
			RequestTimezone:    "UTC",
			ScheduledAt:        time.Now().UTC(),
		}},
	}
	calls := &fakeCallProvider{err: errors.New("sip down")}
	svc := newTestService(nil, welfare, nil, calls)
	_, err := svc.DispatchDueWelfareChecks(context.Background(), 5)
	if err == nil {
		t.Fatal("expected systemic failure when all runs fail")
	}
	if len(welfare.failed) != 1 || welfare.failed[0].NextAttemptAt == nil {
		t.Fatalf("expected retryable failure with next_attempt_at, got %+v", welfare.failed)
	}
	if len(welfare.missed) != 0 {
		t.Fatalf("expected no missed mark on first dial failure")
	}
}

func TestDispatchDialFailureSchedulesRetry(t *testing.T) {
	t.Setenv("WELFARE_CHECK_ALLOW_CONSENT_BYPASS", "true")
	t.Setenv("WELFARE_CHECK_MAX_ATTEMPTS", "5")
	t.Setenv("WELFARE_CHECK_RETRY_BASE_SECONDS", "120")
	runID := uuid.New()
	welfare := &fakeWelfareRepo{
		claimed: []models.WelfareCheckRun{{
			ID:                 runID,
			RequestID:          uuid.New(),
			PatientID:          uuid.New(),
			Attempts:           2,
			PatientPhoneNumber: "+15551234567",
			PatientUserID:      uuid.New(),
			RequestReasonCode:  models.WelfareReasonOther,
			RequestTimezone:    "UTC",
			ScheduledAt:        time.Now().UTC(),
		}},
	}
	calls := &fakeCallProvider{err: errors.New("sip status: 603: Declined")}
	svc := newTestService(nil, welfare, nil, calls)
	_, err := svc.DispatchDueWelfareChecks(context.Background(), 5)
	if err == nil {
		t.Fatal("expected dial error")
	}
	if len(welfare.failed) != 1 || welfare.failed[0].NextAttemptAt == nil {
		t.Fatalf("expected scheduled retry, got %+v", welfare.failed)
	}
	delay := time.Until(*welfare.failed[0].NextAttemptAt)
	// attempt 2 => 2 * base = 4 minutes (± slack)
	if delay < 3*time.Minute || delay > 5*time.Minute {
		t.Fatalf("expected ~4m backoff, got %v", delay)
	}
}

func TestDispatchDialFailureExhaustsToMissed(t *testing.T) {
	t.Setenv("WELFARE_CHECK_ALLOW_CONSENT_BYPASS", "true")
	t.Setenv("WELFARE_CHECK_MAX_ATTEMPTS", "3")
	welfare := &fakeWelfareRepo{
		claimed: []models.WelfareCheckRun{{
			ID:                 uuid.New(),
			RequestID:          uuid.New(),
			PatientID:          uuid.New(),
			Attempts:           3,
			PatientPhoneNumber: "+15551234567",
			PatientUserID:      uuid.New(),
			RequestReasonCode:  models.WelfareReasonOther,
			RequestTimezone:    "UTC",
			ScheduledAt:        time.Now().UTC(),
		}},
	}
	calls := &fakeCallProvider{err: errors.New("sip status: 603: Declined")}
	svc := newTestService(nil, welfare, nil, calls)
	_, err := svc.DispatchDueWelfareChecks(context.Background(), 5)
	if err == nil {
		t.Fatal("expected dial error")
	}
	if len(welfare.missed) != 1 {
		t.Fatalf("expected missed after max attempts, got failed=%+v missed=%v", welfare.failed, welfare.missed)
	}
	if len(welfare.failed) != 0 {
		t.Fatalf("expected no pending retry after exhaustion, got %+v", welfare.failed)
	}
}

func TestDispatchMissingPhoneIsPermanent(t *testing.T) {
	t.Setenv("WELFARE_CHECK_ALLOW_CONSENT_BYPASS", "true")
	welfare := &fakeWelfareRepo{
		claimed: []models.WelfareCheckRun{{
			ID:                 uuid.New(),
			RequestID:          uuid.New(),
			PatientID:          uuid.New(),
			Attempts:           1,
			PatientPhoneNumber: "",
			PatientUserID:      uuid.New(),
			RequestReasonCode:  models.WelfareReasonOther,
			RequestTimezone:    "UTC",
			ScheduledAt:        time.Now().UTC(),
		}},
	}
	svc := newTestService(nil, welfare, nil, &fakeCallProvider{})
	_, err := svc.DispatchDueWelfareChecks(context.Background(), 5)
	if err == nil {
		t.Fatal("expected phone required error")
	}
	if len(welfare.failed) != 1 || welfare.failed[0].NextAttemptAt != nil {
		t.Fatalf("expected permanent failure, got %+v", welfare.failed)
	}
}
