package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	"github.com/KoiralaSam/ZorbaHealth/shared/meetingjoin"
	schedpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/scheduling"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ScheduleMeetingRequest struct {
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	PatientID       string `json:"patient_id,omitempty"`
	StartsAt        string `json:"starts_at"`
	DurationMinutes int32  `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title,omitempty"`
	Notes           string `json:"notes,omitempty"`
	SendSMS         bool   `json:"send_sms,omitempty"`
	CorrelationID   string `json:"correlation_id,omitempty"`
}

type MeetingResponse struct {
	ID              string `json:"id"`
	PatientID       string `json:"patient_id"`
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	StartsAt        string `json:"starts_at"`
	DurationMinutes int32  `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title"`
	JoinURL              string `json:"join_url"`
	Status               string `json:"status"`
	CorrelationID        string `json:"correlation_id"`
	LiveKitRoomName      string `json:"livekit_room_name,omitempty"`
	LiveKitServerURL     string `json:"livekit_server_url,omitempty"`
	ParticipantToken     string `json:"participant_token,omitempty"`
}

type ListMeetingsResponse struct {
	Meetings []MeetingResponse `json:"meetings"`
}

type CancelMeetingRequest struct {
	Reason string `json:"reason,omitempty"`
}

type RescheduleMeetingRequest struct {
	StartsAt        string `json:"starts_at"`
	DurationMinutes int32  `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title,omitempty"`
}

type ListSchedulableStaffResponse struct {
	Staff []StaffSummaryResponse `json:"staff"`
}

type StaffSummaryResponse struct {
	StaffID    string `json:"staff_id"`
	HospitalID string `json:"hospital_id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Email      string `json:"email"`
}

func PatientScheduleMeetingHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var body ScheduleMeetingRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: err.Error()})
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()

	start, err := time.Parse(time.RFC3339, strings.TrimSpace(body.StartsAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "starts_at must be RFC3339"})
		return
	}
	corr := strings.TrimSpace(body.CorrelationID)
	if corr == "" {
		corr = uuid.NewString()
	}
	resp, err := client.ScheduleHealthStaffMeeting(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.ScheduleHealthStaffMeetingRequest{
		PatientId:       claims.PatientID,
		StaffId:         strings.TrimSpace(body.StaffID),
		HospitalId:      strings.TrimSpace(body.HospitalID),
		StartsAt:        timestamppb.New(start),
		DurationMinutes: body.DurationMinutes,
		Timezone:        strings.TrimSpace(body.Timezone),
		Title:           body.Title,
		Notes:           body.Notes,
		Channel:         "portal",
		CorrelationId:   corr,
		SendSms:         body.SendSMS,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "SCHEDULE_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusCreated, map[string]any{"meeting": meetingFromProtoForPatient(resp.GetMeeting())}, nil)
}

func PatientListMeetingsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListScheduledMeetings(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.ListScheduledMeetingsRequest{
		PatientId: claims.PatientID,
		Limit:     50,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, ListMeetingsResponse{Meetings: meetingsFromProtoForPatient(resp.GetMeetings())}, nil)
}

func PatientCancelMeetingHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	meetingID := r.PathValue("id")
	if meetingID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST", Message: "meeting id required"})
		return
	}
	var body CancelMeetingRequest
	_ = decodeJSON(r, &body)
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.CancelScheduledMeeting(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.CancelScheduledMeetingRequest{
		MeetingId: meetingID,
		Reason:    body.Reason,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "CANCEL_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"meeting": meetingFromProtoForPatient(resp.GetMeeting())}, nil)
}

func PatientListSchedulableStaffHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	hospitalID := strings.TrimSpace(r.URL.Query().Get("hospital_id"))
	if hospitalID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST", Message: "hospital_id query required"})
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListSchedulableStaff(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.ListSchedulableStaffRequest{
		PatientId:  claims.PatientID,
		HospitalId: hospitalID,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "LIST_STAFF_FAILED", Message: err.Error()})
		return
	}
	staff := make([]StaffSummaryResponse, 0, len(resp.GetStaff()))
	for _, s := range resp.GetStaff() {
		staff = append(staff, StaffSummaryResponse{
			StaffID: s.GetStaffId(), HospitalID: s.GetHospitalId(), Name: s.GetName(), Role: s.GetRole(), Email: s.GetEmail(),
		})
	}
	writeJson(w, http.StatusOK, ListSchedulableStaffResponse{Staff: staff}, nil)
}

func HospitalScheduleMeetingHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var body ScheduleMeetingRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: err.Error()})
		return
	}
	if strings.TrimSpace(body.PatientID) == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "patient_id is required"})
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(body.StartsAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "starts_at must be RFC3339"})
		return
	}
	hospitalID := strings.TrimSpace(body.HospitalID)
	if hospitalID == "" {
		hospitalID = claims.HospitalID
	}
	// Hospital dashboards often omit staff_id; assign to the authenticated clinician.
	staffID := strings.TrimSpace(body.StaffID)
	if staffID == "" {
		staffID = strings.TrimSpace(claims.StaffID)
	}
	if staffID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "staff_id is required"})
		return
	}
	corr := strings.TrimSpace(body.CorrelationID)
	if corr == "" {
		corr = uuid.NewString()
	}
	resp, err := client.ScheduleHealthStaffMeeting(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.ScheduleHealthStaffMeetingRequest{
		PatientId:       strings.TrimSpace(body.PatientID),
		StaffId:         staffID,
		HospitalId:      hospitalID,
		StartsAt:        timestamppb.New(start),
		DurationMinutes: body.DurationMinutes,
		Timezone:        strings.TrimSpace(body.Timezone),
		Title:           body.Title,
		Notes:           body.Notes,
		Channel:         "dashboard",
		CorrelationId:   corr,
		SendSms:         body.SendSMS,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "SCHEDULE_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusCreated, map[string]any{"meeting": meetingFromProtoForStaff(resp.GetMeeting())}, nil)
}

func HospitalListMeetingsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListScheduledMeetings(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.ListScheduledMeetingsRequest{
		HospitalId: claims.HospitalID,
		Limit:      100,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, ListMeetingsResponse{Meetings: meetingsFromProtoForStaff(resp.GetMeetings())}, nil)
}

func HospitalAcceptMeetingHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	meetingID := r.PathValue("id")
	if meetingID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST", Message: "meeting id required"})
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.AcceptScheduledMeeting(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.AcceptScheduledMeetingRequest{
		MeetingId: meetingID,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "ACCEPT_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"meeting": meetingFromProtoForStaff(resp.GetMeeting())}, nil)
}

func HospitalRescheduleMeetingHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	meetingID := r.PathValue("id")
	if meetingID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST", Message: "meeting id required"})
		return
	}
	var body RescheduleMeetingRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: err.Error()})
		return
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(body.StartsAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "starts_at must be RFC3339"})
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.RescheduleScheduledMeeting(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.RescheduleScheduledMeetingRequest{
		MeetingId:       meetingID,
		StartsAt:        timestamppb.New(start),
		DurationMinutes: body.DurationMinutes,
		Timezone:        strings.TrimSpace(body.Timezone),
		Title:           body.Title,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "RESCHEDULE_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"meeting": meetingFromProtoForStaff(resp.GetMeeting())}, nil)
}

func HospitalCancelMeetingHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	meetingID := r.PathValue("id")
	var body CancelMeetingRequest
	_ = decodeJSON(r, &body)
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.CancelScheduledMeeting(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.CancelScheduledMeetingRequest{
		MeetingId: meetingID,
		Reason:    body.Reason,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "CANCEL_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"meeting": meetingFromProtoForStaff(resp.GetMeeting())}, nil)
}

func meetingFromProtoForPatient(m *schedpb.ScheduledMeeting) MeetingResponse {
	return meetingFromProtoForRole(m, meetingJoinRolePatient)
}

func meetingFromProtoForStaff(m *schedpb.ScheduledMeeting) MeetingResponse {
	return meetingFromProtoForRole(m, meetingJoinRoleStaff)
}

func meetingFromProtoForRole(m *schedpb.ScheduledMeeting, role meetingJoinRole) MeetingResponse {
	if m == nil {
		return MeetingResponse{}
	}
	start := ""
	if m.GetStartsAt() != nil {
		start = m.GetStartsAt().AsTime().Format(time.RFC3339)
	}
	room := strings.TrimSpace(m.GetLivekitRoomName())
	if room == "" {
		room = meetingjoin.RoomFromJoinURL(m.GetJoinUrl())
	}
	var token string
	switch role {
	case meetingJoinRolePatient:
		token = m.GetPatientToken()
	case meetingJoinRoleStaff:
		token = m.GetStaffToken()
	}
	serverWS := meetingjoin.LiveKitServerURL(m.GetJoinUrl())
	if pub := strings.TrimSpace(os.Getenv("LIVEKIT_PUBLIC_WS_URL")); pub != "" {
		serverWS = pub
	}
	return MeetingResponse{
		ID: m.GetId(), PatientID: m.GetPatientId(), StaffID: m.GetStaffId(), HospitalID: m.GetHospitalId(),
		StartsAt: start, DurationMinutes: m.GetDurationMinutes(), Timezone: m.GetTimezone(),
		Title: m.GetTitle(), JoinURL: meetingJoinURLForRole(m, role), Status: m.GetStatus(), CorrelationID: m.GetCorrelationId(),
		LiveKitRoomName: room, LiveKitServerURL: serverWS, ParticipantToken: token,
	}
}

func meetingsFromProtoForPatient(in []*schedpb.ScheduledMeeting) []MeetingResponse {
	return meetingsFromProtoForRole(in, meetingJoinRolePatient)
}

func meetingsFromProtoForStaff(in []*schedpb.ScheduledMeeting) []MeetingResponse {
	return meetingsFromProtoForRole(in, meetingJoinRoleStaff)
}

func meetingsFromProtoForRole(in []*schedpb.ScheduledMeeting, role meetingJoinRole) []MeetingResponse {
	out := make([]MeetingResponse, 0, len(in))
	for _, m := range in {
		out = append(out, meetingFromProtoForRole(m, role))
	}
	return out
}

func decodeJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dest)
}
