package services

import (
	"context"
	"errors"
	"strings"

	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/ports/outbound"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
)

var _ inbound.ProviderService = (*ProviderService)(nil)

type ProviderService struct {
	hospitals outbound.HospitalRepository
	auth      outbound.AuthRepository
}

func NewProviderService(hospitals outbound.HospitalRepository, auth outbound.AuthRepository) *ProviderService {
	return &ProviderService{hospitals: hospitals, auth: auth}
}

func (s *ProviderService) RegisterHospital(ctx context.Context, req models.HospitalRegistration) (*models.HospitalStaffAccount, error) {
	hospitalName := strings.TrimSpace(req.HospitalName)
	licenseNo := strings.TrimSpace(req.LicenseNo)
	staffName := strings.TrimSpace(req.StaffName)
	staffEmail := strings.TrimSpace(strings.ToLower(req.StaffEmail))
	staffPhone, err := normalizePhone(req.StaffPhone)
	if err != nil {
		return nil, err
	}
	staffRole := normalizeStaffRole(req.StaffRole)
	if hospitalName == "" || licenseNo == "" || staffName == "" || staffEmail == "" || strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("hospital_name, license_no, staff_name, email, and password are required")
	}

	userID, err := s.auth.RegisterHospitalStaffUser(ctx, staffEmail, staffPhone, req.Password)
	if err != nil {
		return nil, err
	}

	account, err := s.hospitals.CreateHospitalWithStaff(ctx, models.CreateHospitalStaffInput{
		HospitalName: hospitalName,
		LicenseNo:    licenseNo,
		UserID:       userID,
		StaffName:    staffName,
		StaffEmail:   staffEmail,
		StaffPhone:   staffPhone,
		StaffRole:    staffRole,
	})
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (s *ProviderService) RegisterHospitalStaff(ctx context.Context, req models.HospitalStaffRegistration) (*models.HospitalStaffAccount, error) {
	hospitalID := strings.TrimSpace(req.HospitalID)
	staffName := strings.TrimSpace(req.StaffName)
	staffEmail := strings.TrimSpace(strings.ToLower(req.StaffEmail))
	staffPhone, err := normalizePhone(req.StaffPhone)
	if err != nil {
		return nil, err
	}
	staffRole := normalizeStaffRole(req.StaffRole)
	if hospitalID == "" || staffName == "" || staffEmail == "" || strings.TrimSpace(req.Password) == "" {
		return nil, errors.New("hospital_id, staff_name, email, and password are required")
	}

	userID, err := s.auth.RegisterHospitalStaffUser(ctx, staffEmail, staffPhone, req.Password)
	if err != nil {
		return nil, err
	}

	return s.hospitals.CreateStaffForHospital(ctx, models.CreateHospitalStaffInput{
		HospitalID: hospitalID,
		UserID:     userID,
		StaffName:  staffName,
		StaffEmail: staffEmail,
		StaffPhone: staffPhone,
		StaffRole:  staffRole,
	})
}

func normalizeStaffRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "doctor", "nurse", "billing":
		return strings.TrimSpace(strings.ToLower(role))
	default:
		return "admin"
	}
}

func normalizePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return "", nil
	}
	return sharedauth.ValidatePhoneForStorage(phone)
}
