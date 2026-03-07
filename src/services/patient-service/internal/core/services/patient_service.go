package services

import (
	"context"
	"errors"
	"regexp"
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
}

func NewPatientService(repo outbound.PatientRepository, authService outbound.AuthRepository, pendingRegistrationRepo outbound.PendingRegistrationRepository) *PatientService {
	return &PatientService{repo: repo, authService: authService, pendingRegistrationRepo: pendingRegistrationRepo}
}

func (s *PatientService) StartRegistrationWithVerification(ctx context.Context, req *models.RegisterPatientRequest) (verificationToken string, err error) {
	if req == nil {
		return "", errors.New("registration request is required")
	}
	if !phoneRegex.MatchString(req.PhoneNumber) {
		return "", errors.New("invalid phone number: must be 10–15 digits, optional leading +")
	}
	if req.DateOfBirth.IsZero() {
		return "", errors.New("date of birth is required")
	}
	if req.DateOfBirth.After(time.Now()) {
		return "", errors.New("date of birth cannot be in the future")
	}

	token := uuid.New().String()
	pendingRegistration := &models.PendingRegistration{
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Password:    req.Password,
		FullName:    req.FullName,
		DateOfBirth: req.DateOfBirth,
		CreatedAt:   time.Now(),
	}
	ttl := 15 * time.Minute
	if err := s.pendingRegistrationRepo.Set(ctx, token, pendingRegistration, ttl); err != nil {
		return "", errors.New("failed to set pending registration: " + err.Error())
	}
	return token, nil
}

func (s *PatientService) VerifyEmailAndCreatePatient(ctx context.Context, token string) (*models.Patient, error) {
	pending, err := s.pendingRegistrationRepo.Get(ctx, token)
	if err != nil {
		return nil, errors.New("invalid or expired verification link: " + err.Error())
	}
	defer s.pendingRegistrationRepo.Delete(ctx, token)
	req := &models.RegisterPatientRequest{
		PhoneNumber: pending.PhoneNumber,
		Email:       pending.Email,
		Password:    pending.Password,
		FullName:    pending.FullName,
		DateOfBirth: pending.DateOfBirth,
	}

	// Create user in auth service
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
	return patient, nil
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

func (s *PatientService) GetPatientByPhoneNumber(
	ctx context.Context,
	phoneNumber string,
) (*models.Patient, error) {
	// business rules can live here
	// e.g., normalize phone number format before lookup
	return s.repo.GetPatientByPhoneNumber(ctx, phoneNumber)
}

func (s *PatientService) GetPatientByEmail(
	ctx context.Context,
	email string,
) (*models.Patient, error) {
	// business rules can live here
	// e.g., normalize email to lowercase before lookup
	return s.repo.GetPatientByEmail(ctx, email)
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
