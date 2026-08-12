package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	appointmentpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/appointment"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BookAppointmentRequest struct {
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	PatientID       string `json:"patient_id,omitempty"`
	StartsAt        string `json:"starts_at"`
	DurationMinutes int32  `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Type            string `json:"type,omitempty"`
	Title           string `json:"title,omitempty"`
	Notes           string `json:"notes,omitempty"`
	SendSMS         bool   `json:"send_sms,omitempty"`
	SendEmail       bool   `json:"send_email,omitempty"`
	CorrelationID   string `json:"correlation_id,omitempty"`
}

type AppointmentResponse struct {
	ID              string `json:"id"`
	PatientID       string `json:"patient_id"`
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
	StartsAt        string `json:"starts_at"`
	DurationMinutes int32  `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	Channel         string `json:"channel"`
	Title           string `json:"title"`
	JoinURL         string `json:"join_url,omitempty"`
	CorrelationID   string `json:"correlation_id"`
	LiveKitRoomName string `json:"livekit_room_name,omitempty"`
	ParticipantToken string `json:"participant_token,omitempty"`
}

type AppointmentSlotResponse struct {
	StartsAt        string `json:"starts_at"`
	EndsAt          string `json:"ends_at"`
	DurationMinutes int32  `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	StaffID         string `json:"staff_id"`
	HospitalID      string `json:"hospital_id"`
}

type AvailabilityRuleRequest struct {
	Weekday             int32  `json:"weekday"`
	StartTimeLocal      string `json:"start_time_local"`
	EndTimeLocal        string `json:"end_time_local"`
	SlotDurationMinutes int32  `json:"slot_duration_minutes"`
	Timezone            string `json:"timezone"`
	EffectiveFrom       string `json:"effective_from,omitempty"`
	EffectiveUntil      string `json:"effective_until,omitempty"`
}

type SetAvailabilityRequest struct {
	StaffID    string                    `json:"staff_id,omitempty"`
	HospitalID string                    `json:"hospital_id,omitempty"`
	Rules      []AvailabilityRuleRequest `json:"rules"`
}

type AvailabilityExceptionRequest struct {
	StaffID     string `json:"staff_id,omitempty"`
	HospitalID  string `json:"hospital_id,omitempty"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at"`
	Reason      string `json:"reason,omitempty"`
	IsAvailable bool   `json:"is_available,omitempty"`
}

type RescheduleAppointmentRequestBody struct {
	StartsAt        string `json:"starts_at"`
	DurationMinutes int32  `json:"duration_minutes"`
	Timezone        string `json:"timezone"`
	Title           string `json:"title,omitempty"`
}

func newAppointmentClientFromEnv() (appointmentpb.AppointmentServiceClient, *grpc.ClientConn, error) {
	addr := os.Getenv("APPOINTMENT_SERVICE_GRPC_ADDR")
	if addr == "" {
		addr = "appointment-service:9099"
	}
	conn, err := grpcclient.Dial(addr)
	if err != nil {
		return nil, nil, err
	}
	return appointmentpb.NewAppointmentServiceClient(conn), conn, nil
}

func appointmentFromProto(a *appointmentpb.Appointment, forPatient bool) AppointmentResponse {
	if a == nil {
		return AppointmentResponse{}
	}
	out := AppointmentResponse{
		ID:              a.GetId(),
		PatientID:       a.GetPatientId(),
		StaffID:         a.GetStaffId(),
		HospitalID:      a.GetHospitalId(),
		StartsAt:        a.GetStartsAt().AsTime().Format(time.RFC3339),
		DurationMinutes: a.GetDurationMinutes(),
		Timezone:        a.GetTimezone(),
		Type:            a.GetType(),
		Status:          a.GetStatus(),
		Channel:         a.GetChannel(),
		Title:           a.GetTitle(),
		JoinURL:         a.GetJoinUrl(),
		CorrelationID:   a.GetCorrelationId(),
		LiveKitRoomName: a.GetLivekitRoomName(),
	}
	if forPatient {
		out.ParticipantToken = a.GetPatientToken()
	} else {
		out.ParticipantToken = a.GetStaffToken()
	}
	return out
}

func PatientListAppointmentSlotsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	staffID := strings.TrimSpace(r.URL.Query().Get("staff_id"))
	hospitalID := strings.TrimSpace(r.URL.Query().Get("hospital_id"))
	if staffID == "" || hospitalID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST", Message: "staff_id and hospital_id required"})
		return
	}
	from := time.Now().UTC()
	to := from.AddDate(0, 0, 14)
	if v := strings.TrimSpace(r.URL.Query().Get("from")); v != "" {
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			from = parsed
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("to")); v != "" {
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			to = parsed
		}
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListAvailableSlots(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.ListAvailableSlotsRequest{
		StaffId:    staffID,
		HospitalId: hospitalID,
		From:       timestamppb.New(from),
		To:         timestamppb.New(to),
		Limit:      100,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "SLOTS_FAILED", Message: err.Error()})
		return
	}
	slots := make([]AppointmentSlotResponse, 0, len(resp.GetSlots()))
	for _, s := range resp.GetSlots() {
		slots = append(slots, AppointmentSlotResponse{
			StartsAt:        s.GetStartsAt().AsTime().Format(time.RFC3339),
			EndsAt:          s.GetEndsAt().AsTime().Format(time.RFC3339),
			DurationMinutes: s.GetDurationMinutes(),
			Timezone:        s.GetTimezone(),
			StaffID:         s.GetStaffId(),
			HospitalID:      s.GetHospitalId(),
		})
	}
	writeJson(w, http.StatusOK, map[string]any{"slots": slots}, nil)
}

func PatientBookAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var body BookAppointmentRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: err.Error()})
		return
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(body.StartsAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "starts_at must be RFC3339"})
		return
	}
	corr := strings.TrimSpace(body.CorrelationID)
	if corr == "" {
		corr = uuid.NewString()
	}
	typ := strings.TrimSpace(body.Type)
	if typ == "" {
		typ = "video"
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.BookAppointment(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.BookAppointmentRequest{
		PatientId:         claims.PatientID,
		StaffId:           strings.TrimSpace(body.StaffID),
		HospitalId:        strings.TrimSpace(body.HospitalID),
		StartsAt:          timestamppb.New(start),
		DurationMinutes:   body.DurationMinutes,
		Timezone:          strings.TrimSpace(body.Timezone),
		Type:              typ,
		Channel:           "portal",
		Title:             body.Title,
		Notes:             body.Notes,
		CorrelationId:     corr,
		BookedByActorType: "patient",
		BookedByActorId:   claims.PatientID,
		SendSms:           body.SendSMS,
		SendEmail:         body.SendEmail,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "BOOK_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusCreated, map[string]any{"appointment": appointmentFromProto(resp.GetAppointment(), true)}, nil)
}

func PatientListAppointmentsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListAppointments(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.ListAppointmentsRequest{
		PatientId: claims.PatientID,
		Limit:     50,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	items := make([]AppointmentResponse, 0, len(resp.GetAppointments()))
	for _, a := range resp.GetAppointments() {
		items = append(items, appointmentFromProto(a, true))
	}
	writeJson(w, http.StatusOK, map[string]any{"appointments": items}, nil)
}

func PatientRescheduleAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	id := r.PathValue("id")
	var body RescheduleAppointmentRequestBody
	if err := decodeJSON(r, &body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: err.Error()})
		return
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(body.StartsAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "starts_at must be RFC3339"})
		return
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.RescheduleAppointment(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.RescheduleAppointmentRequest{
		AppointmentId:   id,
		StartsAt:        timestamppb.New(start),
		DurationMinutes: body.DurationMinutes,
		Timezone:        body.Timezone,
		Title:           body.Title,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "RESCHEDULE_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"appointment": appointmentFromProto(resp.GetAppointment(), true)}, nil)
}

func PatientCancelAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	id := r.PathValue("id")
	var body CancelMeetingRequest
	_ = decodeJSON(r, &body)
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.CancelAppointment(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.CancelAppointmentRequest{
		AppointmentId: id,
		Reason:        body.Reason,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "CANCEL_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"appointment": appointmentFromProto(resp.GetAppointment(), true)}, nil)
}

func HospitalListAppointmentsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListAppointments(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.ListAppointmentsRequest{
		HospitalId: claims.HospitalID,
		StaffId:    claims.StaffID,
		Limit:      100,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	items := make([]AppointmentResponse, 0, len(resp.GetAppointments()))
	for _, a := range resp.GetAppointments() {
		items = append(items, appointmentFromProto(a, false))
	}
	writeJson(w, http.StatusOK, map[string]any{"appointments": items}, nil)
}

func HospitalBookAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var body BookAppointmentRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: err.Error()})
		return
	}
	if strings.TrimSpace(body.PatientID) == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "patient_id required"})
		return
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(body.StartsAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "starts_at must be RFC3339"})
		return
	}
	staffID := strings.TrimSpace(body.StaffID)
	if staffID == "" {
		staffID = claims.StaffID
	}
	hospitalID := strings.TrimSpace(body.HospitalID)
	if hospitalID == "" {
		hospitalID = claims.HospitalID
	}
	typ := strings.TrimSpace(body.Type)
	if typ == "" {
		typ = "video"
	}
	corr := strings.TrimSpace(body.CorrelationID)
	if corr == "" {
		corr = uuid.NewString()
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.BookAppointment(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.BookAppointmentRequest{
		PatientId:         strings.TrimSpace(body.PatientID),
		StaffId:           staffID,
		HospitalId:        hospitalID,
		StartsAt:          timestamppb.New(start),
		DurationMinutes:   body.DurationMinutes,
		Timezone:          strings.TrimSpace(body.Timezone),
		Type:              typ,
		Channel:           "dashboard",
		Title:             body.Title,
		Notes:             body.Notes,
		CorrelationId:     corr,
		BookedByActorType: "staff",
		BookedByActorId:   claims.StaffID,
		SendSms:           body.SendSMS,
		SendEmail:         body.SendEmail,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "BOOK_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusCreated, map[string]any{"appointment": appointmentFromProto(resp.GetAppointment(), false)}, nil)
}

func HospitalGetAvailabilityHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	staffID := strings.TrimSpace(r.URL.Query().Get("staff_id"))
	if staffID == "" {
		staffID = claims.StaffID
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.GetAvailabilityRules(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.GetAvailabilityRulesRequest{
		StaffId:    staffID,
		HospitalId: claims.HospitalID,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "AVAILABILITY_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"rules": resp.GetRules()}, nil)
}

func HospitalSetAvailabilityHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var body SetAvailabilityRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: err.Error()})
		return
	}
	staffID := strings.TrimSpace(body.StaffID)
	if staffID == "" {
		staffID = claims.StaffID
	}
	hospitalID := strings.TrimSpace(body.HospitalID)
	if hospitalID == "" {
		hospitalID = claims.HospitalID
	}
	rules := make([]*appointmentpb.AvailabilityRule, 0, len(body.Rules))
	for _, rule := range body.Rules {
		pr := &appointmentpb.AvailabilityRule{
			Weekday:             rule.Weekday,
			StartTimeLocal:      rule.StartTimeLocal,
			EndTimeLocal:        rule.EndTimeLocal,
			SlotDurationMinutes: rule.SlotDurationMinutes,
			Timezone:            rule.Timezone,
		}
		if rule.EffectiveFrom != "" {
			if t, err := time.Parse(time.RFC3339, rule.EffectiveFrom); err == nil {
				pr.EffectiveFrom = timestamppb.New(t)
			} else if t, err := time.Parse("2006-01-02", rule.EffectiveFrom); err == nil {
				pr.EffectiveFrom = timestamppb.New(t)
			}
		}
		if rule.EffectiveUntil != "" {
			if t, err := time.Parse(time.RFC3339, rule.EffectiveUntil); err == nil {
				pr.EffectiveUntil = timestamppb.New(t)
			} else if t, err := time.Parse("2006-01-02", rule.EffectiveUntil); err == nil {
				pr.EffectiveUntil = timestamppb.New(t)
			}
		}
		rules = append(rules, pr)
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.SetAvailabilityRules(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.SetAvailabilityRulesRequest{
		StaffId:    staffID,
		HospitalId: hospitalID,
		Rules:      rules,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "AVAILABILITY_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"rules": resp.GetRules()}, nil)
}

func HospitalAddAvailabilityExceptionHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var body AvailabilityExceptionRequest
	if err := decodeJSON(r, &body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: err.Error()})
		return
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(body.StartsAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "starts_at must be RFC3339"})
		return
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(body.EndsAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "ends_at must be RFC3339"})
		return
	}
	staffID := strings.TrimSpace(body.StaffID)
	if staffID == "" {
		staffID = claims.StaffID
	}
	hospitalID := strings.TrimSpace(body.HospitalID)
	if hospitalID == "" {
		hospitalID = claims.HospitalID
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.AddAvailabilityException(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.AddAvailabilityExceptionRequest{
		StaffId:     staffID,
		HospitalId:  hospitalID,
		StartsAt:    timestamppb.New(start),
		EndsAt:      timestamppb.New(end),
		Reason:      body.Reason,
		IsAvailable: body.IsAvailable,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "EXCEPTION_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusCreated, map[string]any{"exception": resp.GetException()}, nil)
}

func HospitalDeleteAvailabilityExceptionHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	id := r.PathValue("id")
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	_, err = client.RemoveAvailabilityException(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.RemoveAvailabilityExceptionRequest{
		ExceptionId: id,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "EXCEPTION_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"removed": true}, nil)
}

func HospitalListAppointmentSlotsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	staffID := strings.TrimSpace(r.URL.Query().Get("staff_id"))
	if staffID == "" {
		staffID = claims.StaffID
	}
	hospitalID := strings.TrimSpace(r.URL.Query().Get("hospital_id"))
	if hospitalID == "" {
		hospitalID = claims.HospitalID
	}
	from := time.Now().UTC()
	to := from.AddDate(0, 0, 14)
	if v := strings.TrimSpace(r.URL.Query().Get("from")); v != "" {
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			from = parsed
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("to")); v != "" {
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			to = parsed
		}
	}
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListAvailableSlots(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.ListAvailableSlotsRequest{
		StaffId:    staffID,
		HospitalId: hospitalID,
		From:       timestamppb.New(from),
		To:         timestamppb.New(to),
		Limit:      100,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "SLOTS_FAILED", Message: err.Error()})
		return
	}
	slots := make([]AppointmentSlotResponse, 0, len(resp.GetSlots()))
	for _, s := range resp.GetSlots() {
		slots = append(slots, AppointmentSlotResponse{
			StartsAt:        s.GetStartsAt().AsTime().Format(time.RFC3339),
			EndsAt:          s.GetEndsAt().AsTime().Format(time.RFC3339),
			DurationMinutes: s.GetDurationMinutes(),
			Timezone:        s.GetTimezone(),
			StaffID:         s.GetStaffId(),
			HospitalID:      s.GetHospitalId(),
		})
	}
	writeJson(w, http.StatusOK, map[string]any{"slots": slots}, nil)
}

func HospitalCancelAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	id := r.PathValue("id")
	var body CancelMeetingRequest
	_ = decodeJSON(r, &body)
	client, conn, err := newAppointmentClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.CancelAppointment(grpcclient.WithForwardedToken(r.Context(), accessToken), &appointmentpb.CancelAppointmentRequest{
		AppointmentId: id,
		Reason:        body.Reason,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "CANCEL_FAILED", Message: err.Error()})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"appointment": appointmentFromProto(resp.GetAppointment(), false)}, nil)
}
