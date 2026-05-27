package grpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/core/ports/inbound"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	auditportalpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/auditportal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	auditpb.UnimplementedAuditServiceServer
	auditportalpb.UnimplementedAuditPortalServiceServer
	service inbound.AuditService
}

func NewHandler(server *grpc.Server, service inbound.AuditService) *Handler {
	h := &Handler{service: service}
	auditpb.RegisterAuditServiceServer(server, h)
	auditportalpb.RegisterAuditPortalServiceServer(server, h)
	return h
}

func (h *Handler) AppendAuditEvent(ctx context.Context, req *auditpb.AppendAuditEventRequest) (*auditpb.AppendAuditEventResponse, error) {
	if req == nil || req.GetEvent() == nil {
		return nil, status.Error(codes.InvalidArgument, "event required")
	}
	event, err := h.service.AppendAuditEvent(ctx, fromProtoEvent(req.GetEvent()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditpb.AppendAuditEventResponse{Event: toProtoEvent(event)}, nil
}

func (h *Handler) QueryAuditEvents(ctx context.Context, req *auditpb.QueryAuditEventsRequest) (*auditpb.QueryAuditEventsResponse, error) {
	if err := requireAuditorOrAdmin(ctx); err != nil {
		return nil, err
	}
	events, err := h.service.QueryAuditEvents(ctx, models.AuditEventFilter{
		EventType:     req.GetEventType(),
		ActorType:     req.GetActorType(),
		ActorID:       req.GetActorId(),
		PatientID:     req.GetPatientId(),
		ServiceName:   req.GetServiceName(),
		CorrelationID: req.GetCorrelationId(),
		Limit:         req.GetLimit(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditpb.QueryAuditEventsResponse{Events: toProtoEvents(events)}, nil
}

func (h *Handler) CheckConsent(ctx context.Context, req *auditpb.CheckConsentRequest) (*auditpb.CheckConsentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	allowed, denial, consent, err := h.service.CheckConsent(ctx, req.GetPatientId(), req.GetConsentType(), req.GetScope())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditpb.CheckConsentResponse{
		Allowed:      allowed,
		DenialReason: denial,
		Consent:      toProtoConsentPtr(consent),
	}, nil
}

func (h *Handler) GrantConsent(ctx context.Context, req *auditpb.GrantConsentRequest) (*auditpb.GrantConsentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	consent, err := h.service.GrantConsent(ctx, models.Consent{
		PatientID:      req.GetPatientId(),
		ConsentType:    req.GetConsentType(),
		GrantedBy:      grantedByFromClaims(claims, req.GetGrantedBy()),
		GrantedAt:      time.Now().UTC(),
		Scope:          req.GetScope(),
		ExpirationTime: timestampPtr(req.GetExpirationTime()),
		Source:         req.GetSource(),
		MetadataJSON:   structToJSON(req.GetMetadata()),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditpb.GrantConsentResponse{Consent: toProtoConsent(consent)}, nil
}

func (h *Handler) RevokeConsent(ctx context.Context, req *auditpb.RevokeConsentRequest) (*auditpb.RevokeConsentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if _, err := sharedauth.ClaimsFromContext(ctx); err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	consent, err := h.service.RevokeConsent(ctx, req.GetPatientId(), req.GetConsentType(), req.GetScope(), req.GetSource(), structToJSON(req.GetMetadata()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditpb.RevokeConsentResponse{Consent: toProtoConsent(consent)}, nil
}

func (h *Handler) ListConsents(ctx context.Context, req *auditpb.ListConsentsRequest) (*auditpb.ListConsentsResponse, error) {
	if err := requireAuditorOrAdmin(ctx); err != nil {
		return nil, err
	}
	consents, err := h.service.ListConsents(ctx, models.ConsentFilter{
		PatientID:      req.GetPatientId(),
		ConsentType:    req.GetConsentType(),
		IncludeRevoked: req.GetIncludeRevoked(),
		Limit:          req.GetLimit(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*auditpb.Consent, 0, len(consents))
	for _, consent := range consents {
		out = append(out, toProtoConsent(consent))
	}
	return &auditpb.ListConsentsResponse{Consents: out}, nil
}

func (h *Handler) ListPatientConsents(ctx context.Context, req *auditportalpb.ListPatientConsentsRequest) (*auditportalpb.ListPatientConsentsResponse, error) {
	claims, err := requirePatientSelf(ctx, req.GetPatientId())
	if err != nil {
		return nil, err
	}
	_ = claims
	consents, err := h.service.ListPatientConsents(ctx, req.GetPatientId(), req.GetIncludeRevoked(), req.GetLimit())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*auditportalpb.PortalConsent, 0, len(consents))
	for _, consent := range consents {
		out = append(out, toPortalConsent(consent))
	}
	return &auditportalpb.ListPatientConsentsResponse{Consents: out}, nil
}

func (h *Handler) ListPatientAuditEvents(ctx context.Context, req *auditportalpb.ListPatientAuditEventsRequest) (*auditportalpb.ListPatientAuditEventsResponse, error) {
	if _, err := requirePatientSelf(ctx, req.GetPatientId()); err != nil {
		return nil, err
	}
	events, err := h.service.ListPatientAuditEvents(ctx, req.GetPatientId(), req.GetLimit())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditportalpb.ListPatientAuditEventsResponse{Events: toPortalEvents(events)}, nil
}

func (h *Handler) ListHospitalIncidents(ctx context.Context, req *auditportalpb.ListHospitalIncidentsRequest) (*auditportalpb.ListHospitalIncidentsResponse, error) {
	if _, err := requireStaffOrAdmin(ctx); err != nil {
		return nil, err
	}
	events, err := h.service.ListHospitalIncidents(ctx, models.HospitalIncidentFilter{
		Limit: req.GetLimit(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditportalpb.ListHospitalIncidentsResponse{Incidents: toPortalEvents(events)}, nil
}

func (h *Handler) ListHospitalPatientAuditEvents(ctx context.Context, req *auditportalpb.ListHospitalPatientAuditEventsRequest) (*auditportalpb.ListHospitalPatientAuditEventsResponse, error) {
	claims, err := requireStaffOrAdmin(ctx)
	if err != nil {
		return nil, err
	}
	allowed, err := h.service.CheckHospitalConsentAccess(ctx, req.GetPatientId(), claims.HospitalID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify patient consent: "+err.Error())
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "the patient has not consented to share data with this hospital")
	}
	events, err := h.service.ListHospitalPatientAuditEvents(ctx, req.GetPatientId(), req.GetLimit())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &auditportalpb.ListHospitalPatientAuditEventsResponse{Events: toPortalEvents(events)}, nil
}

func requireAuditorOrAdmin(ctx context.Context) error {
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, "no verified claims")
	}
	if claims.ActorType == sharedauth.ActorAdmin || claims.Role == "AUDITOR" {
		return nil
	}
	return status.Error(codes.PermissionDenied, "auditor or admin access required")
}

func requirePatientSelf(ctx context.Context, patientID string) (*sharedauth.Claims, error) {
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if claims.ActorType != sharedauth.ActorPatient || claims.PatientID == "" {
		return nil, status.Error(codes.PermissionDenied, "patient access required")
	}
	if claims.PatientID != patientID {
		return nil, status.Error(codes.PermissionDenied, "patient_id mismatch")
	}
	return claims, nil
}

func requireStaffOrAdmin(ctx context.Context) (*sharedauth.Claims, error) {
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}
	if claims.ActorType == sharedauth.ActorAdmin {
		return claims, nil
	}
	if claims.ActorType == sharedauth.ActorStaff && claims.HospitalID != "" {
		return claims, nil
	}
	return nil, status.Error(codes.PermissionDenied, "hospital staff or admin access required")
}

func grantedByFromClaims(claims *sharedauth.Claims, fallback string) string {
	if fallback != "" {
		return fallback
	}
	switch claims.ActorType {
	case sharedauth.ActorPatient:
		return claims.PatientID
	case sharedauth.ActorStaff:
		return claims.StaffID
	case sharedauth.ActorAdmin:
		return claims.AdminID
	default:
		return "system"
	}
}

func fromProtoEvent(event *auditpb.AuditEvent) models.AuditEvent {
	return models.AuditEvent{
		EventID:       event.GetEventId(),
		EventType:     event.GetEventType(),
		ActorType:     event.GetActorType(),
		ActorID:       event.GetActorId(),
		PatientID:     event.GetPatientId(),
		ServiceName:   event.GetServiceName(),
		ResourceType:  event.GetResourceType(),
		ResourceID:    event.GetResourceId(),
		Timestamp:     event.GetTimestamp().AsTime(),
		RequestID:     event.GetRequestId(),
		CorrelationID: event.GetCorrelationId(),
		IPAddress:     event.GetIpAddress(),
		ToolName:      event.GetToolName(),
		ModelName:     event.GetModelName(),
		ProviderName:  event.GetProviderName(),
		SuccessStatus: event.GetSuccessStatus(),
		FailureReason: event.GetFailureReason(),
		MetadataJSON:  structToJSON(event.GetMetadata()),
	}
}

func toProtoEvent(event models.AuditEvent) *auditpb.AuditEvent {
	return &auditpb.AuditEvent{
		EventId:       event.EventID,
		EventType:     event.EventType,
		ActorType:     event.ActorType,
		ActorId:       event.ActorID,
		PatientId:     event.PatientID,
		ServiceName:   event.ServiceName,
		ResourceType:  event.ResourceType,
		ResourceId:    event.ResourceID,
		Timestamp:     timestamppb.New(event.Timestamp),
		RequestId:     event.RequestID,
		CorrelationId: event.CorrelationID,
		IpAddress:     event.IPAddress,
		ToolName:      event.ToolName,
		ModelName:     event.ModelName,
		ProviderName:  event.ProviderName,
		SuccessStatus: event.SuccessStatus,
		FailureReason: event.FailureReason,
		Metadata:      jsonToStruct(event.MetadataJSON),
	}
}

func toProtoEvents(events []models.AuditEvent) []*auditpb.AuditEvent {
	out := make([]*auditpb.AuditEvent, 0, len(events))
	for _, event := range events {
		out = append(out, toProtoEvent(event))
	}
	return out
}

func toPortalEvents(events []models.AuditEvent) []*auditportalpb.PortalAuditEvent {
	out := make([]*auditportalpb.PortalAuditEvent, 0, len(events))
	for _, event := range events {
		out = append(out, &auditportalpb.PortalAuditEvent{
			EventId:       event.EventID,
			EventType:     event.EventType,
			ActorType:     event.ActorType,
			ActorId:       event.ActorID,
			PatientId:     event.PatientID,
			ServiceName:   event.ServiceName,
			ResourceType:  event.ResourceType,
			ResourceId:    event.ResourceID,
			Timestamp:     timestamppb.New(event.Timestamp),
			CorrelationId: event.CorrelationID,
			ToolName:      event.ToolName,
			SuccessStatus: event.SuccessStatus,
			FailureReason: event.FailureReason,
			Metadata:      jsonToStruct(event.MetadataJSON),
		})
	}
	return out
}

func toProtoConsent(consent models.Consent) *auditpb.Consent {
	out := &auditpb.Consent{
		ConsentId:  consent.ConsentID,
		PatientId:  consent.PatientID,
		ConsentType: consent.ConsentType,
		GrantedBy:  consent.GrantedBy,
		GrantedAt:  timestamppb.New(consent.GrantedAt),
		Scope:      consent.Scope,
		Source:     consent.Source,
		Metadata:   jsonToStruct(consent.MetadataJSON),
	}
	if consent.RevokedAt != nil {
		out.RevokedAt = timestamppb.New(*consent.RevokedAt)
	}
	if consent.ExpirationTime != nil {
		out.ExpirationTime = timestamppb.New(*consent.ExpirationTime)
	}
	return out
}

func toPortalConsent(consent models.Consent) *auditportalpb.PortalConsent {
	out := &auditportalpb.PortalConsent{
		ConsentId:   consent.ConsentID,
		PatientId:   consent.PatientID,
		ConsentType: consent.ConsentType,
		GrantedBy:   consent.GrantedBy,
		GrantedAt:   timestamppb.New(consent.GrantedAt),
		Scope:       consent.Scope,
		Source:      consent.Source,
		Metadata:    jsonToStruct(consent.MetadataJSON),
	}
	if consent.RevokedAt != nil {
		out.RevokedAt = timestamppb.New(*consent.RevokedAt)
	}
	if consent.ExpirationTime != nil {
		out.ExpirationTime = timestamppb.New(*consent.ExpirationTime)
	}
	return out
}

func toProtoConsentPtr(consent *models.Consent) *auditpb.Consent {
	if consent == nil {
		return nil
	}
	return toProtoConsent(*consent)
}

func timestampPtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	value := ts.AsTime()
	return &value
}

func structToJSON(s *structpb.Struct) []byte {
	if s == nil {
		return []byte(`{}`)
	}
	data, err := json.Marshal(s.AsMap())
	if err != nil {
		return []byte(`{}`)
	}
	return data
}

func jsonToStruct(data []byte) *structpb.Struct {
	if len(data) == 0 {
		return &structpb.Struct{}
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return &structpb.Struct{}
	}
	s, err := structpb.NewStruct(payload)
	if err != nil {
		return &structpb.Struct{}
	}
	return s
}
