package services

import (
	"context"
	"crypto/rand"
	"regexp"
	"strings"
	"time"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	"github.com/google/uuid"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
)

// E.164-ish: optional +, then 10–15 digits.
var phoneRegex = regexp.MustCompile(`^\+?[0-9]{10,15}$`)

type PatientService struct {
	repo                    outbound.PatientRepository
	authService             outbound.AuthRepository
	pendingRegistrationRepo outbound.PendingRegistrationRepository
	publisher               outbound.PatientPublisher
	auditClient             auditpb.AuditServiceClient
}

func NewPatientService(
	repo outbound.PatientRepository,
	authService outbound.AuthRepository,
	pendingRegistrationRepo outbound.PendingRegistrationRepository,
	publisher outbound.PatientPublisher,
	auditClient auditpb.AuditServiceClient,
) *PatientService {
	return &PatientService{
		repo:                    repo,
		authService:             authService,
		pendingRegistrationRepo: pendingRegistrationRepo,
		publisher:               publisher,
		auditClient:             auditClient,
	}
}

func (s *PatientService) StartRegistrationWithVerification(ctx context.Context, req *models.RegisterPatientRequest) (verificationToken string, otp string, err error) {
	if req == nil {
		return "", "", domainErrors.ErrRegistrationRequestRequired
	}
	if !phoneRegex.MatchString(req.PhoneNumber) {
		return "", "", domainErrors.ErrInvalidPhoneNumber
	}
	req.PhoneNumber = normalizePhone(req.PhoneNumber)
	if req.DateOfBirth.IsZero() {
		return "", "", domainErrors.ErrDateOfBirthRequired
	}
	if req.DateOfBirth.After(time.Now()) {
		return "", "", domainErrors.ErrDateOfBirthInFuture
	}

	otp, err = generateOTP(6)
	if err != nil {
		return "", "", domainErrors.ErrGenerateOTPFailed
	}

	token := uuid.New().String()
	pendingRegistration := &models.PendingRegistration{
		Email:         req.Email,
		PhoneNumber:   req.PhoneNumber,
		Password:      req.Password,
		FullName:      req.FullName,
		DateOfBirth:   req.DateOfBirth,
		CreatedAt:     time.Now(),
		PhoneVerified: false,
		EmailVerified: false,
	}
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, token, pendingRegistration, ttl); err != nil {
		return "", "", domainErrors.ErrPendingRegistrationSetFailed
	}
	otpTTL := 5 * time.Minute
	if err := s.pendingRegistrationRepo.SetOTP(ctx, req.PhoneNumber, token, otp, otpTTL); err != nil {
		return "", "", domainErrors.ErrOTPSetFailed
	}
	if s.publisher != nil {
		if err := s.publisher.PublishPatientChached(ctx, req, token, otp); err != nil {
			return "", "", domainErrors.ErrPublishPatientCachedEventFailed
		}
	}
	return token, otp, nil
}

func generateOTP(digits int) (string, error) {
	const digitset = "0123456789"
	b := make([]byte, digits)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digitset[int(b[i])%len(digitset)]
	}
	return string(b), nil
}

func (s *PatientService) VerifyEmailAndCreatePatient(ctx context.Context, token string) (*models.Patient, error) {
	pending, err := s.pendingRegistrationRepo.Get(ctx, token)
	if err != nil {
		return nil, domainErrors.ErrInvalidOrExpiredVerificationLink
	}

	// Mark email as verified (user clicked the email link)
	pending.EmailVerified = true
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, token, pending, ttl); err != nil {
		return nil, domainErrors.ErrPendingRegistrationUpdateFailed
	}

	if !pending.PhoneVerified {
		return nil, domainErrors.ErrPhoneVerificationRequired
	}

	// Both verified: create patient and clean up
	defer s.pendingRegistrationRepo.Delete(ctx, token)
	req := &models.RegisterPatientRequest{
		PhoneNumber: pending.PhoneNumber,
		Email:       pending.Email,
		Password:    pending.Password,
		FullName:    pending.FullName,
		DateOfBirth: pending.DateOfBirth,
	}

	authResult, err := s.authService.RegisterPatient(ctx, req)
	if err != nil {
		return nil, domainErrors.ErrAuthServiceRegisterPatientFailed
	}

	userID, err := uuid.Parse(authResult.UserID)
	if err != nil {
		return nil, domainErrors.ErrAuthServiceInvalidUserID
	}

	patient := &models.Patient{
		UserID:      userID,
		PhoneNumber: pending.PhoneNumber,
		Email:       pending.Email,
		FullName:    pending.FullName,
		DateOfBirth: pending.DateOfBirth,
	}
	patient, err = s.repo.CreatePatient(ctx, patient)
	if err != nil {
		return nil, domainErrors.ErrPatientCreationFailed
	}
	if s.publisher != nil {
		if err := s.publisher.PublishPatientRegistered(ctx, patient); err != nil {
			return nil, domainErrors.ErrPublishPatientRegisteredEventFailed
		}
	}
	return patient, nil
}

// VerifyPhoneOTP verifies the OTP for the given phone and sets PhoneVerified on the pending registration.
func (s *PatientService) VerifyPhoneOTP(ctx context.Context, phone string, code string) error {
	normalized := normalizePhone(phone)
	if normalized == "" {
		return domainErrors.ErrInvalidPhoneNumberNoDigits
	}

	storedToken, storedCode, err := s.pendingRegistrationRepo.GetOTP(ctx, normalized)
	if err != nil {
		return domainErrors.ErrInvalidOrExpiredOTP
	}
	if storedCode != code {
		return domainErrors.ErrInvalidOTPCode
	}
	if strings.HasPrefix(storedToken, "existing:") {
		return domainErrors.ErrExistingPatientVerificationState
	}

	pending, err := s.pendingRegistrationRepo.Get(ctx, storedToken)
	if err != nil {
		return domainErrors.ErrPendingRegistrationNotFoundOrExpired
	}
	pending.PhoneVerified = true
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, storedToken, pending, ttl); err != nil {
		return domainErrors.ErrPendingRegistrationUpdateFailed
	}
	_ = s.pendingRegistrationRepo.DeleteOTP(ctx, normalized)
	return nil
}

func (s *PatientService) VerifyExistingPhoneOTP(ctx context.Context, phone string, code string) (*models.PatientSessionResult, error) {
	normalized := normalizePhone(phone)
	if normalized == "" {
		return nil, domainErrors.ErrInvalidPhoneNumberNoDigits
	}

	storedToken, storedCode, err := s.pendingRegistrationRepo.GetOTP(ctx, normalized)
	if err != nil {
		return nil, domainErrors.ErrInvalidOrExpiredOTP
	}
	if storedCode != code {
		return nil, domainErrors.ErrInvalidOTPCode
	}
	if !strings.HasPrefix(storedToken, "existing:") {
		return nil, domainErrors.ErrExistingPatientVerificationState
	}

	patientID := strings.TrimPrefix(storedToken, "existing:")
	if patientID == "" {
		return nil, domainErrors.ErrExistingPatientVerificationState
	}

	_ = s.pendingRegistrationRepo.DeleteOTP(ctx, normalized)
	patient, err := s.repo.GetPatientByID(ctx, patientID)
	if err != nil {
		return nil, err
	}
	session, err := s.authService.CreatePatientSession(ctx, patient.UserID.String(), []string{"location:read", "records:read"})
	if err != nil {
		return nil, domainErrors.ErrAuthServiceRegisterPatientFailed
	}
	_ = session
	return &models.PatientSessionResult{
		PatientID:   patient.ID.String(),
		UserID:      patient.UserID.String(),
		AccessToken: session.AccessToken,
	}, nil
}

func (s *PatientService) LoginPatient(
	ctx context.Context,
	patient *models.Patient,
) (*models.PatientSessionResult, error) {
	if patient == nil {
		return nil, domainErrors.ErrRegistrationRequestRequired
	}

	userID, _, err := s.authService.ValidateUserCredentials(ctx, &models.LoginRequest{
		Email:       strings.TrimSpace(patient.Email),
		PhoneNumber: normalizePhone(patient.PhoneNumber),
		Password:    patient.MedicalNotes,
	})
	if err != nil {
		return nil, err
	}

	patientRecord, err := s.repo.GetPatientByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	session, err := s.authService.CreatePatientSession(ctx, patientRecord.UserID.String(), []string{"location:read", "records:read"})
	if err != nil {
		return nil, err
	}

	return &models.PatientSessionResult{
		PatientID:    patientRecord.ID.String(),
		UserID:       patientRecord.UserID.String(),
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
	}, nil
}

func (s *PatientService) GetPatientByID(
	ctx context.Context,
	id string,
) (*models.Patient, error) {
	// business rules can live here
	return s.repo.GetPatientByID(ctx, id)
}

func (s *PatientService) GetPatientProfile(ctx context.Context, patientID string) (*models.PatientProfile, error) {
	return s.repo.GetPatientProfile(ctx, patientID)
}

func (s *PatientService) ListPatientCallSummaries(ctx context.Context, patientID string, limit, offset int32) ([]models.CallSummary, error) {
	return s.repo.ListPatientCallSummaries(ctx, patientID, limit, offset)
}

// normalizePhone returns digits only (E.164 without +) for consistent lookup.
func normalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *PatientService) GetPatientByPhoneNumber(
	ctx context.Context,
	phoneNumber string,
) (*models.Patient, error) {
	normalized := normalizePhone(phoneNumber)
	if normalized == "" {
		return nil, domainErrors.ErrInvalidPhoneNumberNoDigits
	}
	return s.repo.GetPatientByPhoneNumber(ctx, normalized)
}

func (s *PatientService) GetPatientByEmail(
	ctx context.Context,
	email string,
) (*models.Patient, error) {
	normalized := strings.TrimSpace(strings.ToLower(email))
	if normalized == "" {
		return nil, domainErrors.ErrEmailRequired
	}
	return s.repo.GetPatientByEmail(ctx, normalized)
}

func (s *PatientService) UpdatePatient(
	ctx context.Context,
	patient *models.Patient,
) error {
	// business rules can live here
	// e.g., validate updated data, check permissions, audit logging
	return s.repo.UpdatePatient(ctx, patient)
}

func (s *PatientService) DeletePatient(
	ctx context.Context,
	id string,
) error {
	// business rules can live here
	// e.g., check if patient has active calls, cascade delete related data
	return s.repo.DeletePatient(ctx, id)
}
