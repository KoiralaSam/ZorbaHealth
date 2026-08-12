package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/domain/models"
)

// HospitalRepository persists hospitals and staff memberships.
type HospitalRepository interface {
	CreateHospitalWithStaff(ctx context.Context, in models.CreateHospitalStaffInput) (*models.HospitalStaffAccount, error)
	CreateStaffForHospital(ctx context.Context, in models.CreateHospitalStaffInput) (*models.HospitalStaffAccount, error)
}

// AuthRepository creates credential users in auth-service (no hospital domain).
type AuthRepository interface {
	RegisterHospitalStaffUser(ctx context.Context, email, phoneNumber, password string) (userID string, err error)
}
