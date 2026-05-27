package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/adapters/secondary/repositories/postgres/sqlc"
	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/core/domain/models"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{queries: sqlc.New(db)}
}

func (r *Repository) AppendAuditEvent(ctx context.Context, event models.AuditEvent) (models.AuditEvent, error) {
	eventID, err := uuid.Parse(event.EventID)
	if err != nil {
		return models.AuditEvent{}, err
	}

	row, err := r.queries.AppendAuditEvent(ctx, sqlc.AppendAuditEventParams{
		EventID:        pgtype.UUID{Bytes: eventID, Valid: true},
		EventType:      event.EventType,
		ActorType:      event.ActorType,
		ActorID:        event.ActorID,
		PatientID:      event.PatientID,
		ServiceName:    event.ServiceName,
		ResourceType:   event.ResourceType,
		ResourceID:     event.ResourceID,
		EventTimestamp: timestamptz(event.Timestamp),
		RequestID:      event.RequestID,
		CorrelationID:  event.CorrelationID,
		IpAddress:      event.IPAddress,
		ToolName:       event.ToolName,
		ModelName:      event.ModelName,
		ProviderName:   event.ProviderName,
		SuccessStatus:  event.SuccessStatus,
		FailureReason:  event.FailureReason,
		MetadataJson:   event.MetadataJSON,
	})
	if err != nil {
		return models.AuditEvent{}, err
	}
	return auditEventFromRow(row), nil
}

func (r *Repository) QueryAuditEvents(ctx context.Context, filter models.AuditEventFilter) ([]models.AuditEvent, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.queries.QueryAuditEvents(ctx, sqlc.QueryAuditEventsParams{
		EventType:     filter.EventType,
		ActorType:     filter.ActorType,
		ActorID:       filter.ActorID,
		PatientID:     filter.PatientID,
		ServiceName:   filter.ServiceName,
		CorrelationID: filter.CorrelationID,
		ResultLimit:   limit,
	})
	if err != nil {
		return nil, err
	}

	events := make([]models.AuditEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, auditEventFromRow(row))
	}
	return events, nil
}

func (r *Repository) CheckConsent(ctx context.Context, patientID, consentType, scope string) (bool, string, *models.Consent, error) {
	row, err := r.queries.GetLatestMatchingConsent(ctx, sqlc.GetLatestMatchingConsentParams{
		PatientID:   patientID,
		ConsentType: consentType,
		ScopeValue:  scope,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "no matching consent found", nil, nil
		}
		return false, "", nil, err
	}
	consent := consentFromRow(row)

	if consent.RevokedAt != nil {
		return false, "consent has been revoked", &consent, nil
	}
	if consent.ExpirationTime != nil && consent.ExpirationTime.Before(time.Now().UTC()) {
		return false, "consent has expired", &consent, nil
	}
	return true, "", &consent, nil
}

func (r *Repository) CheckHospitalConsentAccess(ctx context.Context, patientID, hospitalID string) (bool, error) {
	patientUUID, err := uuid.Parse(patientID)
	if err != nil {
		return false, err
	}
	hospitalUUID, err := uuid.Parse(hospitalID)
	if err != nil {
		return false, err
	}
	return r.queries.CheckHospitalConsentAccess(ctx, sqlc.CheckHospitalConsentAccessParams{
		PatientID:  pgtype.UUID{Bytes: patientUUID, Valid: true},
		HospitalID: pgtype.UUID{Bytes: hospitalUUID, Valid: true},
	})
}

func (r *Repository) GrantConsent(ctx context.Context, consent models.Consent) (models.Consent, error) {
	consentID, err := uuid.Parse(consent.ConsentID)
	if err != nil {
		return models.Consent{}, err
	}

	row, err := r.queries.GrantConsent(ctx, sqlc.GrantConsentParams{
		ID:             pgtype.UUID{Bytes: consentID, Valid: true},
		PatientID:      consent.PatientID,
		ConsentType:    consent.ConsentType,
		GrantedBy:      consent.GrantedBy,
		GrantedAt:      timestamptz(consent.GrantedAt),
		Scope:          consent.Scope,
		ExpirationTime: optionalTimestamptz(consent.ExpirationTime),
		Source:         consent.Source,
		MetadataJson:   consent.MetadataJSON,
	})
	if err != nil {
		return models.Consent{}, err
	}
	return consentFromRow(row), nil
}

func (r *Repository) RevokeConsent(ctx context.Context, patientID, consentType, scope, source string, metadataJSON []byte) (models.Consent, error) {
	row, err := r.queries.RevokeConsent(ctx, sqlc.RevokeConsentParams{
		Source:       source,
		MetadataJson: metadataJSON,
		PatientID:    patientID,
		ConsentType:  consentType,
		ScopeValue:   scope,
	})
	if err != nil {
		return models.Consent{}, err
	}
	return consentFromRow(row), nil
}

func (r *Repository) ListConsents(ctx context.Context, filter models.ConsentFilter) ([]models.Consent, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.queries.ListConsents(ctx, sqlc.ListConsentsParams{
		PatientID:      filter.PatientID,
		ConsentType:    filter.ConsentType,
		IncludeRevoked: filter.IncludeRevoked,
		ResultLimit:    limit,
	})
	if err != nil {
		return nil, err
	}

	consents := make([]models.Consent, 0, len(rows))
	for _, row := range rows {
		consents = append(consents, consentFromRow(row))
	}
	return consents, nil
}

func (r *Repository) ListPatientConsents(ctx context.Context, patientID string, includeRevoked bool, limit int32) ([]models.Consent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.queries.ListPatientPortalConsents(ctx, sqlc.ListPatientPortalConsentsParams{
		PatientID:      patientID,
		IncludeRevoked: includeRevoked,
		ResultLimit:    limit,
	})
	if err != nil {
		return nil, err
	}
	consents := make([]models.Consent, 0, len(rows))
	for _, row := range rows {
		consents = append(consents, consentFromRow(row))
	}
	return consents, nil
}

func (r *Repository) ListPatientAuditEvents(ctx context.Context, patientID string, limit int32) ([]models.AuditEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.queries.ListPatientPortalAuditEvents(ctx, sqlc.ListPatientPortalAuditEventsParams{
		PatientID:   pgtype.Text{String: patientID, Valid: patientID != ""},
		ResultLimit: limit,
	})
	if err != nil {
		return nil, err
	}
	events := make([]models.AuditEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, auditEventFromRow(row))
	}
	return events, nil
}

func (r *Repository) ListHospitalIncidents(ctx context.Context, filter models.HospitalIncidentFilter) ([]models.AuditEvent, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.queries.ListHospitalPortalIncidents(ctx, sqlc.ListHospitalPortalIncidentsParams{
		EventType:   sharedaudit.EventEmergencyEscalationTriggered,
		ResultLimit: limit,
	})
	if err != nil {
		return nil, err
	}
	events := make([]models.AuditEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, auditEventFromRow(row))
	}
	return events, nil
}

func (r *Repository) ListHospitalPatientAuditEvents(ctx context.Context, patientID string, limit int32) ([]models.AuditEvent, error) {
	return r.ListPatientAuditEvents(ctx, patientID, limit)
}

func auditEventFromRow(row sqlc.AuditAuditEvent) models.AuditEvent {
	return models.AuditEvent{
		EventID:       uuidString(row.EventID),
		EventType:     row.EventType,
		ActorType:     row.ActorType,
		ActorID:       row.ActorID,
		PatientID:     textValue(row.PatientID),
		ServiceName:   row.ServiceName,
		ResourceType:  textValue(row.ResourceType),
		ResourceID:    textValue(row.ResourceID),
		Timestamp:     timeValue(row.EventTimestamp),
		RequestID:     textValue(row.RequestID),
		CorrelationID: textValue(row.CorrelationID),
		IPAddress:     textValue(row.IpAddress),
		ToolName:      textValue(row.ToolName),
		ModelName:     textValue(row.ModelName),
		ProviderName:  textValue(row.ProviderName),
		SuccessStatus: row.SuccessStatus,
		FailureReason: textValue(row.FailureReason),
		MetadataJSON:  row.MetadataJson,
	}
}

func consentFromRow(row sqlc.AuditConsent) models.Consent {
	return models.Consent{
		ConsentID:      uuidString(row.ID),
		PatientID:      row.PatientID,
		ConsentType:    row.ConsentType,
		GrantedBy:      row.GrantedBy,
		GrantedAt:      timeValue(row.GrantedAt),
		RevokedAt:      timePtr(row.RevokedAt),
		Scope:          row.Scope,
		ExpirationTime: timePtr(row.ExpirationTime),
		Source:         row.Source,
		MetadataJSON:   row.MetadataJson,
	}
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func optionalTimestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return timestamptz(*value)
}
