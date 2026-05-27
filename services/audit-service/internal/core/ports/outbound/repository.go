package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/core/domain/models"
)

type Repository interface {
	AppendAuditEvent(ctx context.Context, event models.AuditEvent) (models.AuditEvent, error)
	QueryAuditEvents(ctx context.Context, filter models.AuditEventFilter) ([]models.AuditEvent, error)
	CheckConsent(ctx context.Context, patientID, consentType, scope string) (bool, string, *models.Consent, error)
	CheckHospitalConsentAccess(ctx context.Context, patientID, hospitalID string) (bool, error)
	GrantConsent(ctx context.Context, consent models.Consent) (models.Consent, error)
	RevokeConsent(ctx context.Context, patientID, consentType, scope, source string, metadataJSON []byte) (models.Consent, error)
	ListConsents(ctx context.Context, filter models.ConsentFilter) ([]models.Consent, error)
	ListPatientConsents(ctx context.Context, patientID string, includeRevoked bool, limit int32) ([]models.Consent, error)
	ListPatientAuditEvents(ctx context.Context, patientID string, limit int32) ([]models.AuditEvent, error)
	ListHospitalIncidents(ctx context.Context, filter models.HospitalIncidentFilter) ([]models.AuditEvent, error)
	ListHospitalPatientAuditEvents(ctx context.Context, patientID string, limit int32) ([]models.AuditEvent, error)
}
