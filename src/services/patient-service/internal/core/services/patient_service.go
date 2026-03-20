package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
	"github.com/google/uuid"
)

// E.164-ish: optional +, then 10–15 digits.
var phoneRegex = regexp.MustCompile(`^\+?[0-9]{10,15}$`)

type PatientService struct {
	repo                    outbound.PatientRepository
	authService             outbound.AuthRepository
	pendingRegistrationRepo outbound.PendingRegistrationRepository
	publisher               outbound.PatientPublisher
}

func NewPatientService(
	repo outbound.PatientRepository,
	authService outbound.AuthRepository,
	pendingRegistrationRepo outbound.PendingRegistrationRepository,
	publisher outbound.PatientPublisher,
) *PatientService {
	return &PatientService{
		repo:                    repo,
		authService:             authService,
		pendingRegistrationRepo: pendingRegistrationRepo,
		publisher:               publisher,
	}
}

func (s *PatientService) StartRegistrationWithVerification(ctx context.Context, req *models.RegisterPatientRequest) (verificationToken string, otp string, err error) {
	if req == nil {
		return "", "", errors.New("registration request is required")
	}
	if !phoneRegex.MatchString(req.PhoneNumber) {
		return "", "", errors.New("invalid phone number: must be 10–15 digits, optional leading +")
	}
	if req.DateOfBirth.IsZero() {
		return "", "", errors.New("date of birth is required")
	}
	if req.DateOfBirth.After(time.Now()) {
		return "", "", errors.New("date of birth cannot be in the future")
	}

	otp, err = generateOTP(6)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate OTP: %w", err)
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
		return "", "", errors.New("failed to set pending registration: " + err.Error())
	}
	otpTTL := 5 * time.Minute
	if err := s.pendingRegistrationRepo.SetOTP(ctx, req.PhoneNumber, token, otp, otpTTL); err != nil {
		return "", "", errors.New("failed to set OTP: " + err.Error())
	}
	if s.publisher != nil {
		if err := s.publisher.PublishPatientChached(ctx, req, token, otp); err != nil {
			return "", "", errors.New("failed to publish patient cached event: " + err.Error())
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
		return nil, errors.New("invalid or expired verification link: " + err.Error())
	}

	// Mark email as verified (user clicked the email link)
	pending.EmailVerified = true
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, token, pending, ttl); err != nil {
		return nil, errors.New("failed to update pending registration: " + err.Error())
	}

	if !pending.PhoneVerified {
		return nil, errors.New("verify your phone to complete registration")
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
		return nil, errors.New("failed to create user in auth service: " + err.Error())
	}

	userID, err := uuid.Parse(authResult.UserID)
	if err != nil {
		return nil, errors.New("invalid user_id from auth service: " + err.Error())
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
		return nil, errors.New("failed to create patient: " + err.Error())
	}
	if s.publisher != nil {
		if err := s.publisher.PublishPatientRegistered(ctx, patient); err != nil {
			return nil, errors.New("failed to publish patient registered event: " + err.Error())
		}
	}
	return patient, nil
}

// VerifyPhoneOTP verifies the OTP for the given phone and sets PhoneVerified on the pending registration.
func (s *PatientService) VerifyPhoneOTP(ctx context.Context, phone string, code string) error {
	storedToken, storedCode, err := s.pendingRegistrationRepo.GetOTP(ctx, phone)
	if err != nil {
		return errors.New("invalid or expired OTP")
	}
	if storedCode != code {
		return errors.New("invalid OTP code")
	}

	pending, err := s.pendingRegistrationRepo.Get(ctx, storedToken)
	if err != nil {
		return errors.New("pending registration not found or expired")
	}
	pending.PhoneVerified = true
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, storedToken, pending, ttl); err != nil {
		return errors.New("failed to update pending registration: " + err.Error())
	}
	_ = s.pendingRegistrationRepo.DeleteOTP(ctx, phone)
	return nil
}

func (s *PatientService) LoginPatient(
	ctx context.Context,
	patient *models.Patient,
) (*models.Patient, error) {
	// TODO: patient login is handled by the dedicated auth-service.
	// For now this method is a placeholder to keep the handler signature stable.
	return s.repo.GetPatientByEmail(ctx, patient.Email)
}

func (s *PatientService) GetPatientByID(
	ctx context.Context,
	id string,
) (*models.Patient, error) {
	// business rules can live here
	return s.repo.GetPatientByID(ctx, id)
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
		return nil, errors.New("invalid phone number: no digits")
	}
	return s.repo.GetPatientByPhoneNumber(ctx, normalized)
}

func (s *PatientService) GetPatientByEmail(
	ctx context.Context,
	email string,
) (*models.Patient, error) {
	normalized := strings.TrimSpace(strings.ToLower(email))
	if normalized == "" {
		return nil, errors.New("email is required")
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
