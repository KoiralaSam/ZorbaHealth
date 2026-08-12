package inbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/domain/models"
)

// ProviderService is the inbound port for hospital/organization domain use cases.
type ProviderService interface {
	RegisterHospital(ctx context.Context, req models.HospitalRegistration) (*models.HospitalStaffAccount, error)
	RegisterHospitalStaff(ctx context.Context, req models.HospitalStaffRegistration) (*models.HospitalStaffAccount, error)
}
