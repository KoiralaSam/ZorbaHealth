package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/core/ports/inbound"
	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/core/ports/outbound"
)

type Service struct {
	repo outbound.Repository
}

func New(repo outbound.Repository) inbound.AuditService {
	return &Service{repo: repo}
}

func (s *Service) AppendAuditEvent(ctx context.Context, event models.AuditEvent) (models.AuditEvent, error) {
	if event.EventID == "" {
		event.EventID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if len(event.MetadataJSON) == 0 {
		event.MetadataJSON = []byte(`{}`)
	}
	return s.repo.AppendAuditEvent(ctx, event)
}

func (s *Service) QueryAuditEvents(ctx context.Context, filter models.AuditEventFilter) ([]models.AuditEvent, error) {
	return s.repo.QueryAuditEvents(ctx, filter)
}

func (s *Service) CheckConsent(ctx context.Context, patientID, consentType, scope string) (bool, string, *models.Consent, error) {
	return s.repo.CheckConsent(ctx, patientID, consentType, scope)
}

func (s *Service) GrantConsent(ctx context.Context, consent models.Consent) (models.Consent, error) {
	if consent.ConsentID == "" {
		consent.ConsentID = uuid.NewString()
	}
	if consent.GrantedAt.IsZero() {
		consent.GrantedAt = time.Now().UTC()
	}
	if len(consent.MetadataJSON) == 0 {
		consent.MetadataJSON = []byte(`{}`)
	}
	if !json.Valid(consent.MetadataJSON) {
		consent.MetadataJSON = []byte(`{}`)
	}
	return s.repo.GrantConsent(ctx, consent)
}

func (s *Service) RevokeConsent(ctx context.Context, patientID, consentType, scope, source string, metadataJSON []byte) (models.Consent, error) {
	if len(metadataJSON) == 0 {
		metadataJSON = []byte(`{}`)
	}
	if !json.Valid(metadataJSON) {
		metadataJSON = []byte(`{}`)
	}
	return s.repo.RevokeConsent(ctx, patientID, consentType, scope, source, metadataJSON)
}

func (s *Service) ListConsents(ctx context.Context, filter models.ConsentFilter) ([]models.Consent, error) {
	return s.repo.ListConsents(ctx, filter)
}

func (s *Service) ListPatientConsents(ctx context.Context, patientID string, includeRevoked bool, limit int32) ([]models.Consent, error) {
	return s.repo.ListPatientConsents(ctx, patientID, includeRevoked, limit)
}

func (s *Service) ListPatientAuditEvents(ctx context.Context, patientID string, limit int32) ([]models.AuditEvent, error) {
	return s.repo.ListPatientAuditEvents(ctx, patientID, limit)
}

func (s *Service) ListHospitalIncidents(ctx context.Context, filter models.HospitalIncidentFilter) ([]models.AuditEvent, error) {
	if filter.Limit <= 0 {
		filter.Limit = 30
	}
	return s.repo.ListHospitalIncidents(ctx, filter)
}

func (s *Service) ListHospitalPatientAuditEvents(ctx context.Context, patientID string, limit int32) ([]models.AuditEvent, error) {
	return s.repo.ListHospitalPatientAuditEvents(ctx, patientID, limit)
}

func (s *Service) CheckHospitalConsentAccess(ctx context.Context, patientID, hospitalID string) (bool, error) {
	return s.repo.CheckHospitalConsentAccess(ctx, patientID, hospitalID)
}
