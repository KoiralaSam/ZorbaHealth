package outbound

import (
	"context"

	models "github.com/KoiralaSam/ZorbaHealth/services/analytics-service/internal/core/domain/models"
)

// PatientAnalyticsRepository is for a patient's own call history.
type PatientAnalyticsRepository interface {
	GetPatientCallHistory(ctx context.Context, patientID string, limit int) (*models.PatientCallHistory, error)
}
