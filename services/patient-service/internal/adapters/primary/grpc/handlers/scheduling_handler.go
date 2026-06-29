package grpc

import (
	"context"
	"errors"
	"strings"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	schedpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/scheduling"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ScheduleHealthStaffMeeting(ctx context.Context, req *schedpb.ScheduleHealthStaffMeetingRequest) (*schedpb.ScheduleHealthStaffMeetingResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := authorizeScheduleRequest(claims, req); err != nil {
		return nil, err
	}
	if req.GetStartsAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "starts_at is required")
	}
	correlationID, err := parseUUIDOrNew(req.GetCorrelationId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid correlation_id")
	}
	patientID, err := uuid.Parse(strings.TrimSpace(req.GetPatientId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid patient_id")
	}
	staffID, err := uuid.Parse(strings.TrimSpace(req.GetStaffId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
	}
	hospitalID, err := uuid.Parse(strings.TrimSpace(req.GetHospitalId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
	}

	actorType, actorID := actorFromClaims(claims)
	cmd := &models.ScheduleMeetingCommand{
		PatientID:       patientID,
		StaffID:         staffID,
		HospitalID:      hospitalID,
		StartsAt:        req.GetStartsAt().AsTime(),
		DurationMinutes: req.GetDurationMinutes(),
		Timezone:        strings.TrimSpace(req.GetTimezone()),
		Title:           req.GetTitle(),
		Notes:           req.GetNotes(),
		Channel:         models.MeetingChannel(strings.TrimSpace(req.GetChannel())),
		CorrelationID:   correlationID,
		VoiceSessionID:  strings.TrimSpace(req.GetVoiceSessionId()),
		SendSMS:         req.GetSendSms(),
		ActorType:       actorType,
		ActorID:         actorID,
	}
	if cmd.Timezone == "" {
		cmd.Timezone = "UTC"
	}
	if cmd.Channel == "" {
		cmd.Channel = models.MeetingChannelPortal
	}

	meeting, err := h.scheduling.ScheduleHealthStaffMeeting(ctx, cmd)
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	return &schedpb.ScheduleHealthStaffMeetingResponse{Meeting: toProtoMeeting(meeting)}, nil
}

func (h *gRPCHandler) ListScheduledMeetings(ctx context.Context, req *schedpb.ListScheduledMeetingsRequest) (*schedpb.ListScheduledMeetingsResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	filter := models.ListMeetingsFilter{
		IncludeCancelled: req.GetIncludeCancelled(),
		Limit:            req.GetLimit(),
	}
	switch claims.ActorType {
	case sharedauth.ActorPatient:
		pid, err := uuid.Parse(claims.PatientID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid patient")
		}
		filter.PatientID = &pid
	case sharedauth.ActorStaff:
		if strings.TrimSpace(req.GetHospitalId()) != "" {
			hid, err := uuid.Parse(req.GetHospitalId())
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid hospital_id")
			}
			filter.HospitalID = &hid
		} else if strings.TrimSpace(req.GetStaffId()) != "" {
			sid, err := uuid.Parse(req.GetStaffId())
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid staff_id")
			}
			filter.StaffID = &sid
		} else if claims.StaffID != "" {
			sid, err := uuid.Parse(claims.StaffID)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid staff")
			}
			filter.StaffID = &sid
		} else {
			hid, err := uuid.Parse(claims.HospitalID)
			if err != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid hospital")
			}
			filter.HospitalID = &hid
		}
	default:
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}
	meetings, err := h.scheduling.ListScheduledMeetings(ctx, filter)
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	out := make([]*schedpb.ScheduledMeeting, 0, len(meetings))
	for i := range meetings {
		out = append(out, toProtoMeeting(&meetings[i]))
	}
	return &schedpb.ListScheduledMeetingsResponse{Meetings: out}, nil
}

func (h *gRPCHandler) AcceptScheduledMeeting(ctx context.Context, req *schedpb.AcceptScheduledMeetingRequest) (*schedpb.AcceptScheduledMeetingResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	actorType, actorID := actorFromClaims(claims)
	actor := models.ScheduleActor{
		ActorType:  actorType,
		ActorID:    actorID,
		PatientID:  claims.PatientID,
		StaffID:    claims.StaffID,
		HospitalID: claims.HospitalID,
	}
	meeting, err := h.scheduling.AcceptScheduledMeeting(ctx, strings.TrimSpace(req.GetMeetingId()), actor)
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	return &schedpb.AcceptScheduledMeetingResponse{Meeting: toProtoMeeting(meeting)}, nil
}

func (h *gRPCHandler) RescheduleScheduledMeeting(ctx context.Context, req *schedpb.RescheduleScheduledMeetingRequest) (*schedpb.RescheduleScheduledMeetingResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetStartsAt() == nil {
		return nil, status.Error(codes.InvalidArgument, "starts_at is required")
	}
	actorType, actorID := actorFromClaims(claims)
	actor := models.ScheduleActor{
		ActorType:  actorType,
		ActorID:    actorID,
		PatientID:  claims.PatientID,
		StaffID:    claims.StaffID,
		HospitalID: claims.HospitalID,
	}
	meeting, err := h.scheduling.RescheduleScheduledMeeting(ctx, strings.TrimSpace(req.GetMeetingId()), actor, req.GetStartsAt().AsTime(), req.GetDurationMinutes(), req.GetTimezone(), req.GetTitle())
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	return &schedpb.RescheduleScheduledMeetingResponse{Meeting: toProtoMeeting(meeting)}, nil
}

func (h *gRPCHandler) CancelScheduledMeeting(ctx context.Context, req *schedpb.CancelScheduledMeetingRequest) (*schedpb.CancelScheduledMeetingResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	actorType, actorID := actorFromClaims(claims)
	actor := models.ScheduleActor{
		ActorType:  actorType,
		ActorID:    actorID,
		PatientID:  claims.PatientID,
		StaffID:    claims.StaffID,
		HospitalID: claims.HospitalID,
	}
	meeting, err := h.scheduling.CancelScheduledMeeting(ctx, req.GetMeetingId(), actor, req.GetReason())
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	return &schedpb.CancelScheduledMeetingResponse{Meeting: toProtoMeeting(meeting)}, nil
}

func (h *gRPCHandler) ListSchedulableStaff(ctx context.Context, req *schedpb.ListSchedulableStaffRequest) (*schedpb.ListSchedulableStaffResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if claims.ActorType != sharedauth.ActorPatient || claims.PatientID == "" {
		return nil, status.Error(codes.PermissionDenied, "patient access required")
	}
	if claims.PatientID != strings.TrimSpace(req.GetPatientId()) {
		return nil, status.Error(codes.PermissionDenied, "forbidden")
	}
	staff, err := h.scheduling.ListSchedulableStaff(ctx, req.GetPatientId(), req.GetHospitalId())
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	out := make([]*schedpb.StaffSummary, 0, len(staff))
	for _, s := range staff {
		out = append(out, &schedpb.StaffSummary{
			StaffId:    s.StaffID.String(),
			HospitalId: s.HospitalID.String(),
			Name:       s.Name,
			Role:       s.Role,
			Email:      s.Email,
		})
	}
	return &schedpb.ListSchedulableStaffResponse{Staff: out}, nil
}

func (h *gRPCHandler) RequestBridgedCallTransfer(ctx context.Context, req *schedpb.RequestBridgedCallTransferRequest) (*schedpb.RequestBridgedCallTransferResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	accessToken, _ := grpcclient.ForwardedTokenFromContext(ctx)
	actorType, actorID := actorFromClaims(claims)
	if claims.ActorType != sharedauth.ActorPatient || claims.PatientID != strings.TrimSpace(req.GetPatientId()) {
		return nil, status.Error(codes.PermissionDenied, "patient access required")
	}
	session, err := h.scheduling.RequestBridgedCallTransfer(ctx, &models.RequestBridgedCallTransferCommand{
		SessionID:   strings.TrimSpace(req.GetSessionId()),
		RoomSID:     strings.TrimSpace(req.GetRoomSid()),
		PatientID:   strings.TrimSpace(req.GetPatientId()),
		HospitalID:  strings.TrimSpace(req.GetHospitalId()),
		StaffID:     strings.TrimSpace(req.GetStaffId()),
		Reason:      strings.TrimSpace(req.GetTransferReason()),
		ActorType:   actorType,
		ActorID:     actorID,
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	resp := &schedpb.RequestBridgedCallTransferResponse{
		Session: toProtoBridgedCallSession(session),
	}
	if token, err := h.scheduling.MintBridgedCallPatientToken(ctx, session); err == nil && token != nil {
		resp.PatientRoomToken = token.Token
		resp.LivekitWsUrl = token.WSURL
	}
	return resp, nil
}

func (h *gRPCHandler) ConnectBridgedCall(ctx context.Context, req *schedpb.ConnectBridgedCallRequest) (*schedpb.ConnectBridgedCallResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	accessToken, _ := grpcclient.ForwardedTokenFromContext(ctx)
	actorType, actorID := actorFromClaims(claims)
	actor := models.ScheduleActor{
		ActorType:  actorType,
		ActorID:    actorID,
		PatientID:  claims.PatientID,
		StaffID:    claims.StaffID,
		HospitalID: claims.HospitalID,
	}
	result, err := h.scheduling.ConnectBridgedCall(ctx, strings.TrimSpace(req.GetSessionId()), actor, strings.TrimSpace(req.GetStaffParticipantIdentity()), accessToken)
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	return &schedpb.ConnectBridgedCallResponse{
		Session:          toProtoBridgedCallSession(result.Session),
		StaffRoomToken:   result.StaffRoomToken,
		LivekitWsUrl:     result.LiveKitWSURL,
		PatientRoomToken: "",
	}, nil
}

func (h *gRPCHandler) ListBridgedCallSessions(ctx context.Context, req *schedpb.ListBridgedCallSessionsRequest) (*schedpb.ListBridgedCallSessionsResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if claims.ActorType != sharedauth.ActorStaff || strings.TrimSpace(claims.HospitalID) == "" {
		return nil, status.Error(codes.PermissionDenied, "hospital staff access required")
	}
	actorType, actorID := actorFromClaims(claims)
	actor := models.ScheduleActor{
		ActorType:  actorType,
		ActorID:    actorID,
		StaffID:    claims.StaffID,
		HospitalID: claims.HospitalID,
	}
	sessions, err := h.scheduling.ListBridgedCallSessions(ctx, actor, strings.TrimSpace(req.GetStatus()), int(req.GetLimit()))
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	out := make([]*schedpb.BridgedCallSession, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, toProtoBridgedCallSession(session))
	}
	return &schedpb.ListBridgedCallSessionsResponse{Sessions: out}, nil
}

func (h *gRPCHandler) GetBridgedCallSession(ctx context.Context, req *schedpb.GetBridgedCallSessionRequest) (*schedpb.GetBridgedCallSessionResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	actorType, actorID := actorFromClaims(claims)
	actor := models.ScheduleActor{
		ActorType:  actorType,
		ActorID:    actorID,
		PatientID:  claims.PatientID,
		StaffID:    claims.StaffID,
		HospitalID: claims.HospitalID,
	}
	session, err := h.scheduling.GetBridgedCallSession(ctx, strings.TrimSpace(req.GetSessionId()), actor)
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	resp := &schedpb.GetBridgedCallSessionResponse{
		Session: toProtoBridgedCallSession(session),
	}
	if claims.ActorType == sharedauth.ActorPatient {
		if token, err := h.scheduling.MintBridgedCallPatientToken(ctx, session); err == nil && token != nil {
			resp.PatientRoomToken = token.Token
			resp.LivekitWsUrl = token.WSURL
		}
	}
	return resp, nil
}

func (h *gRPCHandler) UpdateBridgedCallTranslation(ctx context.Context, req *schedpb.UpdateBridgedCallTranslationRequest) (*schedpb.UpdateBridgedCallTranslationResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	accessToken, _ := grpcclient.ForwardedTokenFromContext(ctx)
	actorType, actorID := actorFromClaims(claims)
	session, err := h.scheduling.UpdateBridgedCallTranslation(ctx, &models.UpdateBridgedCallTranslationCommand{
		SessionID:   strings.TrimSpace(req.GetSessionId()),
		Participant: models.BridgedCallParticipant(strings.TrimSpace(req.GetParticipant())),
		Preferences: fromProtoBridgedCallTranslation(req.GetTranslation()),
		ActorType:   actorType,
		ActorID:     actorID,
		StaffID:     claims.StaffID,
		PatientID:   claims.PatientID,
		HospitalID:  claims.HospitalID,
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	return &schedpb.UpdateBridgedCallTranslationResponse{Session: toProtoBridgedCallSession(session)}, nil
}

func (h *gRPCHandler) EndBridgedCall(ctx context.Context, req *schedpb.EndBridgedCallRequest) (*schedpb.EndBridgedCallResponse, error) {
	claims, err := claimsFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	actorType, actorID := actorFromClaims(claims)
	actor := models.ScheduleActor{
		ActorType:  actorType,
		ActorID:    actorID,
		PatientID:  claims.PatientID,
		StaffID:    claims.StaffID,
		HospitalID: claims.HospitalID,
	}
	session, err := h.scheduling.EndBridgedCall(ctx, strings.TrimSpace(req.GetSessionId()), actor, strings.TrimSpace(req.GetReason()))
	if err != nil {
		return nil, mapSchedulingError(err)
	}
	return &schedpb.EndBridgedCallResponse{Session: toProtoBridgedCallSession(session)}, nil
}

func claimsFromCtx(ctx context.Context) (*sharedauth.Claims, error) {
	token, ok := grpcclient.ForwardedTokenFromContext(ctx)
	if !ok || strings.TrimSpace(token) == "" {
		return nil, status.Error(codes.Unauthenticated, "missing forwarded token")
	}
	claims, err := sharedauth.VerifyToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	return claims, nil
}

func actorFromClaims(claims *sharedauth.Claims) (string, string) {
	switch claims.ActorType {
	case sharedauth.ActorStaff:
		if claims.StaffID != "" {
			return sharedauth.ActorStaff, claims.StaffID
		}
		return sharedauth.ActorStaff, claims.HospitalID
	case sharedauth.ActorPatient:
		return sharedauth.ActorPatient, claims.PatientID
	default:
		return claims.ActorType, claims.PatientID
	}
}

func authorizeScheduleRequest(claims *sharedauth.Claims, req *schedpb.ScheduleHealthStaffMeetingRequest) error {
	patientID := strings.TrimSpace(req.GetPatientId())
	hospitalID := strings.TrimSpace(req.GetHospitalId())
	switch claims.ActorType {
	case sharedauth.ActorPatient:
		if claims.PatientID != patientID {
			return status.Error(codes.PermissionDenied, "forbidden")
		}
	case sharedauth.ActorStaff:
		if claims.HospitalID != hospitalID {
			return status.Error(codes.PermissionDenied, "forbidden")
		}
	default:
		return status.Error(codes.PermissionDenied, "forbidden")
	}
	return nil
}

func parseUUIDOrNew(raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.New(), nil
	}
	return uuid.Parse(raw)
}

func toProtoMeeting(m *models.ScheduledMeeting) *schedpb.ScheduledMeeting {
	if m == nil {
		return nil
	}
	return &schedpb.ScheduledMeeting{
		Id:              m.ID.String(),
		PatientId:       m.PatientID.String(),
		StaffId:         m.StaffID.String(),
		HospitalId:      m.HospitalID.String(),
		StartsAt:        timestamppb.New(m.StartsAt),
		DurationMinutes: m.DurationMinutes,
		Timezone:        m.Timezone,
		Title:           m.Title,
		JoinUrl:         m.JoinURL,
		Status:          string(m.Status),
		CorrelationId:   m.CorrelationID.String(),
		LivekitRoomName: m.LiveKitRoomName,
		PatientToken:    m.PatientToken,
		StaffToken:      m.StaffToken,
	}
}

func toProtoBridgedCallSession(session *models.BridgedCallSession) *schedpb.BridgedCallSession {
	if session == nil {
		return nil
	}
	return &schedpb.BridgedCallSession{
		SessionId:            session.SessionID,
		RoomSid:              session.RoomSID,
		PatientId:            session.PatientID,
		HospitalId:           session.HospitalID,
		StaffId:              session.StaffID,
		Status:               string(session.Status),
		RequestedByActorType: session.RequestedByActorType,
		RequestedByActorId:   session.RequestedByActorID,
		TransferReason:       session.TransferReason,
		RequestedAt:          timestamppb.New(session.RequestedAt),
		ConnectedAt:          timeToProto(session.ConnectedAt),
		EndedAt:              timeToProto(session.EndedAt),
		PatientTranslation:   toProtoBridgedCallTranslation(session.PatientTranslation),
		StaffTranslation:     toProtoBridgedCallTranslation(session.StaffTranslation),
	}
}

func toProtoBridgedCallTranslation(p models.BridgedCallTranslationPreferences) *schedpb.BridgedCallTranslationPreferences {
	return &schedpb.BridgedCallTranslationPreferences{
		Enabled:             p.Enabled,
		LanguageMode:        string(p.LanguageMode),
		LanguageCode:        p.LanguageCode,
		ParticipantIdentity: p.ParticipantIdentity,
		UpdatedAt:           timeToProto(&p.UpdatedAt),
	}
}

func fromProtoBridgedCallTranslation(p *schedpb.BridgedCallTranslationPreferences) models.BridgedCallTranslationPreferences {
	if p == nil {
		return models.BridgedCallTranslationPreferences{}
	}
	return models.BridgedCallTranslationPreferences{
		Enabled:             p.GetEnabled(),
		LanguageMode:        models.TranslationMode(strings.TrimSpace(p.GetLanguageMode())),
		LanguageCode:        strings.TrimSpace(p.GetLanguageCode()),
		ParticipantIdentity: strings.TrimSpace(p.GetParticipantIdentity()),
	}
}

func unwrapSchedulingMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func mapSchedulingError(err error) error {
	switch {
	case errors.Is(err, domainErrors.ErrMeetingStaffNotFound),
		errors.Is(err, domainErrors.ErrMeetingNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domainErrors.ErrMeetingConsentRequired),
		errors.Is(err, domainErrors.ErrMeetingNotificationConsent),
		errors.Is(err, domainErrors.ErrMeetingPatientEmailMissing),
		errors.Is(err, domainErrors.ErrMeetingHospitalMismatch):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domainErrors.ErrMeetingStartsAtInvalid),
		errors.Is(err, domainErrors.ErrMeetingDurationInvalid),
		errors.Is(err, domainErrors.ErrMeetingInvalidRole):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domainErrors.ErrMeetingLiveKitUnavailable),
		errors.Is(err, domainErrors.ErrBridgedCallStoreUnavailable):
		return status.Error(codes.Unavailable, unwrapSchedulingMessage(err))
	case errors.Is(err, domainErrors.ErrMeetingAlreadyCancelled),
		errors.Is(err, domainErrors.ErrMeetingNotPending):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domainErrors.ErrBridgedCallNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domainErrors.ErrBridgedCallForbidden),
		errors.Is(err, domainErrors.ErrMeetingConsentRequired):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domainErrors.ErrBridgedCallSessionRequired),
		errors.Is(err, domainErrors.ErrBridgedCallHospitalRequired),
		errors.Is(err, domainErrors.ErrBridgedCallInvalidMode),
		errors.Is(err, domainErrors.ErrBridgedCallInvalidParticipant):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domainErrors.ErrBridgedCallAlreadyEnded):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		msg := err.Error()
		if strings.Contains(msg, "INTERNAL_SERVICE_SECRET") {
			return status.Error(codes.Unavailable, "scheduling backend misconfigured: patient-service needs INTERNAL_SERVICE_SECRET for audit-service gRPC")
		}
		return status.Error(codes.Internal, msg)
	}
}
