package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/google/uuid"
)

type WelfareCheckRepository interface {
	InsertWelfareCheck(ctx context.Context, check *models.WelfareCheck) (*models.WelfareCheck, error)
	ListWelfareChecks(ctx context.Context, filter models.ListWelfareChecksFilter) ([]models.WelfareCheck, error)
	CancelWelfareCheck(ctx context.Context, patientID, checkID uuid.UUID) (*models.WelfareCheck, error)
	ClaimDueWelfareCheckRuns(ctx context.Context, limit int32) ([]models.WelfareCheckRun, error)
	PersistWelfareRunLiveKitResult(ctx context.Context, result models.WelfareCheckDispatchResult) error
	MarkWelfareRunDispatched(ctx context.Context, result models.WelfareCheckDispatchResult) error
	MarkWelfareRunFailed(ctx context.Context, runID uuid.UUID, reason string, retry bool) error
	MarkWelfareRunMissed(ctx context.Context, runID uuid.UUID, reason string) error
	UpdateWelfareRunLifecycle(ctx context.Context, patientID, runID uuid.UUID, status models.WelfareCheckRunStatus, reason string) (*models.WelfareCheckRun, error)
	GetWelfareCheckRun(ctx context.Context, patientID, runID uuid.UUID) (*models.WelfareCheckRun, error)
}
