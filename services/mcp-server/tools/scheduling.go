package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	schedpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/scheduling"
)

type patientHospitalEntry struct {
	HospitalID   string `json:"hospital_id"`
	HospitalName string `json:"hospital_name"`
}

func queryActivePatientHospitals(ctx context.Context, db *pgxpool.Pool, patientID string) ([]patientHospitalEntry, error) {
	rows, err := db.Query(ctx, `
		SELECT h.id::text, h.name
		FROM patient_hospital_consents c
		INNER JOIN hospitals h ON h.id = c.hospital_id
		WHERE c.patient_id = $1::uuid
		  AND c.revoked_at IS NULL
		ORDER BY h.name
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]patientHospitalEntry, 0)
	for rows.Next() {
		var entry patientHospitalEntry
		if err := rows.Scan(&entry.HospitalID, &entry.HospitalName); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return out, rows.Err()
}

func patientHasActiveHospitalConsent(ctx context.Context, db *pgxpool.Pool, patientID, hospitalID string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM patient_hospital_consents
			WHERE patient_id = $1::uuid
			  AND hospital_id = $2::uuid
			  AND revoked_at IS NULL
		)
	`, patientID, hospitalID).Scan(&ok)
	return ok, err
}

func schedulingRequirements(ctx context.Context, db *pgxpool.Pool, authToken, patientID string) map[string]any {
	req := map[string]any{
		"profile_email_present":      false,
		"email_notification_consent": false,
	}
	if db == nil {
		return req
	}
	var email string
	if err := db.QueryRow(ctx, `
		SELECT COALESCE(TRIM(email::text), '')
		FROM patients
		WHERE id = $1::uuid
	`, patientID).Scan(&email); err == nil {
		req["profile_email_present"] = strings.TrimSpace(email) != ""
	}
	if auditClient != nil && strings.TrimSpace(authToken) != "" {
		auditCtx := ctxWithForwardedToken(ctx, authToken)
		resp, err := auditClient.CheckConsent(auditCtx, &auditpb.CheckConsentRequest{
			PatientId:   patientID,
			ConsentType: sharedaudit.ConsentEmailNotification,
			Scope:       "",
		})
		if err == nil {
			req["email_notification_consent"] = resp.GetAllowed()
		}
	}
	return req
}

func hospitalConsentMismatchMessage(
	ctx context.Context,
	db *pgxpool.Pool,
	patientID, hospitalID string,
	active []patientHospitalEntry,
) string {
	ok, err := patientHasActiveHospitalConsent(ctx, db, patientID, hospitalID)
	if err == nil && ok {
		return ""
	}
	var revokedForHospital bool
	_ = db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM patient_hospital_consents
			WHERE patient_id = $1::uuid
			  AND hospital_id = $2::uuid
			  AND revoked_at IS NOT NULL
		)
	`, patientID, hospitalID).Scan(&revokedForHospital)

	if revokedForHospital {
		return fmt.Sprintf(
			"hospital consent for hospital_id %s was revoked; re-approve in the patient portal (Consents → Hospital consent)",
			hospitalID,
		)
	}
	if len(active) == 0 {
		return "no active hospital consents for this verified patient (patient_hospital_consents with revoked_at NULL)"
	}
	names := make([]string, 0, len(active))
	for _, h := range active {
		names = append(names, fmt.Sprintf("%s (%s)", h.HospitalName, h.HospitalID))
	}
	return fmt.Sprintf(
		"hospital_id %s is not among this patient's active hospital consents. Use one of: %s",
		hospitalID,
		strings.Join(names, "; "),
	)
}

type scheduleHealthStaffMeetingInput struct {
	StaffID          string `json:"staffID" jsonschema:"hospital staff UUID"`
	StartsAt         string `json:"startsAt" jsonschema:"RFC3339 start time"`
	DurationMinutes  int32  `json:"durationMinutes" jsonschema:"meeting length in minutes"`
	Timezone         string `json:"timezone" jsonschema:"IANA timezone"`
	Title            string `json:"title,omitempty"`
	HospitalID       string `json:"hospitalID" jsonschema:"hospital UUID from list_patient_hospitals"`
	PatientConfirmed bool   `json:"patientConfirmed" jsonschema:"true only after the caller verbally confirmed hospital, staff member, date, and time"`
	SendSMS          bool   `json:"sendSms,omitempty"`
	Auth             string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

type listSchedulableStaffInput struct {
	HospitalID string `json:"hospitalID" jsonschema:"hospital UUID from list_patient_hospitals"`
	Auth       string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

type listPatientHospitalsInput struct {
	Auth string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterListPatientHospitals(s *mcp.Server, db *pgxpool.Pool) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_patient_hospitals",
		Description: "List Zorba hospitals the verified patient has active data-sharing consent with; use before scheduling a video visit (not the same as find_nearest_hospital)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in listPatientHospitalsInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
			auditCompat(db, claims, "list_patient_hospitals", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if claims.PatientID == "" || strings.HasPrefix(claims.PatientID, "session:") {
			auditCompat(db, claims, "list_patient_hospitals", "forbidden", "verified patient required")
			return errorResult("identity verification is required before listing hospitals"), nil, nil
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "list_patient_hospitals", map[string]any{
			"session_id": claims.SessionID,
		})

		out, err := queryActivePatientHospitals(ctx, db, claims.PatientID)
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "list_patient_hospitals", "error", err.Error(), correlationID, nil)
			return errorResult("failed to list hospitals"), nil, nil
		}

		payload := map[string]any{
			"patient_id":              claims.PatientID,
			"hospitals":               out,
			"scheduling_requirements": schedulingRequirements(ctx, db, in.Auth, claims.PatientID),
		}
		if len(out) == 0 {
			var activeRows, revokedRows int
			_ = db.QueryRow(ctx, `
				SELECT COUNT(*) FROM patient_hospital_consents
				WHERE patient_id = $1::uuid AND revoked_at IS NULL
			`, claims.PatientID).Scan(&activeRows)
			_ = db.QueryRow(ctx, `
				SELECT COUNT(*) FROM patient_hospital_consents
				WHERE patient_id = $1::uuid AND revoked_at IS NOT NULL
			`, claims.PatientID).Scan(&revokedRows)
			switch {
			case activeRows > 0:
				payload["message"] = fmt.Sprintf(
					"Found %d active consent row(s) for patient_id %s but no matching hospitals table record; check hospital_id FK.",
					activeRows, claims.PatientID,
				)
			case revokedRows > 0:
				payload["message"] = "Hospital consent exists but is revoked (revoked_at set). Re-approve in the patient portal."
			default:
				payload["message"] = fmt.Sprintf(
					"No rows in patient_hospital_consents for verified patient_id %s. Grant hospital consent in the portal (Consents → Hospital consent). If you expected a row, confirm patient_id matches the voice-verified account, not a different login.",
					claims.PatientID,
				)
			}
		} else {
			payload["message"] = "Hospital data-sharing consent is active for the listed facilities. For scheduling, also ensure scheduling_requirements (profile email and email notification consent) are true."
		}
		body, _ := json.Marshal(payload)
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "list_patient_hospitals", "success", "", correlationID, map[string]any{
			"count": len(out),
		})
		return textResult(string(body)), nil, nil
	})
}

func RegisterListSchedulableStaff(s *mcp.Server, db *pgxpool.Pool, client schedpb.SchedulingServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_schedulable_staff",
		Description: "List doctors and nurses at a Zorba hospital for video visit scheduling; hospitalID must come from list_patient_hospitals",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in listSchedulableStaffInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
			auditCompat(db, claims, "list_schedulable_staff", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if claims.PatientID == "" || strings.HasPrefix(claims.PatientID, "session:") {
			auditCompat(db, claims, "list_schedulable_staff", "forbidden", "verified patient required")
			return errorResult("identity verification is required before listing staff"), nil, nil
		}
		if strings.TrimSpace(in.HospitalID) == "" {
			return errorResult("hospitalID is required"), nil, nil
		}
		hospitalID := strings.TrimSpace(in.HospitalID)
		activeHospitals, err := queryActivePatientHospitals(ctx, db, claims.PatientID)
		if err != nil {
			return errorResult("failed to verify hospital consent"), nil, nil
		}
		if msg := hospitalConsentMismatchMessage(ctx, db, claims.PatientID, hospitalID, activeHospitals); msg != "" {
			return errorResult(msg), nil, nil
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "list_schedulable_staff", map[string]any{
			"session_id":  claims.SessionID,
			"hospital_id": in.HospitalID,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.ListSchedulableStaff(ctx, &schedpb.ListSchedulableStaffRequest{
			PatientId:  claims.PatientID,
			HospitalId: hospitalID,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "list_schedulable_staff", "error", err.Error(), correlationID, nil)
			msg := grpcErrorMessage(err, "failed to list staff")
			if strings.Contains(msg, "patient has not consented to share data with this hospital") {
				if hint := hospitalConsentMismatchMessage(ctx, db, claims.PatientID, hospitalID, activeHospitals); hint != "" {
					msg = hint
				}
			}
			return errorResult(msg), nil, nil
		}

		staff := resp.GetStaff()
		type staffEntry struct {
			StaffID string `json:"staff_id"`
			Name    string `json:"name"`
			Role    string `json:"role"`
			Email   string `json:"email,omitempty"`
		}
		out := make([]staffEntry, 0, len(staff))
		for _, s := range staff {
			out = append(out, staffEntry{
				StaffID: s.GetStaffId(),
				Name:    s.GetName(),
				Role:    s.GetRole(),
				Email:   s.GetEmail(),
			})
		}
		body, _ := json.Marshal(out)
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "list_schedulable_staff", "success", "", correlationID, map[string]any{
			"count": len(staff),
		})
		return textResult(string(body)), nil, nil
	})
}

func RegisterScheduleHealthStaffMeeting(s *mcp.Server, db *pgxpool.Pool, client schedpb.SchedulingServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "schedule_health_staff_meeting",
		Description: "Request a video visit with hospital staff after the patient confirmed hospital, staff, date, and time; staff must accept before LiveKit visit links are created",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in scheduleHealthStaffMeetingInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
			auditCompat(db, claims, "schedule_health_staff_meeting", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if claims.PatientID == "" || strings.HasPrefix(claims.PatientID, "session:") {
			auditCompat(db, claims, "schedule_health_staff_meeting", "forbidden", "verified patient required")
			return errorResult("identity verification is required before scheduling"), nil, nil
		}
		if !sharedauth.HasScope(claims, "records:read") {
			auditCompat(db, claims, "schedule_health_staff_meeting", "forbidden", "missing records:read")
			return errorResult("forbidden: missing records:read"), nil, nil
		}
		if !in.PatientConfirmed {
			return errorResult("patientConfirmed must be true after you verbally confirmed hospital, staff member, date, and time with the caller"), nil, nil
		}
		if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.HospitalID) == "" {
			return errorResult("staffID and hospitalID are required"), nil, nil
		}
		hospitalID := strings.TrimSpace(in.HospitalID)
		activeHospitals, err := queryActivePatientHospitals(ctx, db, claims.PatientID)
		if err != nil {
			return errorResult("failed to verify hospital consent"), nil, nil
		}
		if msg := hospitalConsentMismatchMessage(ctx, db, claims.PatientID, hospitalID, activeHospitals); msg != "" {
			return errorResult(msg), nil, nil
		}
		start, err := time.Parse(time.RFC3339, strings.TrimSpace(in.StartsAt))
		if err != nil {
			return errorResult("startsAt must be RFC3339 with timezone offset or Z (example: 2026-06-10T15:30:00-04:00)"), nil, nil
		}
		now := time.Now().UTC()
		if !start.After(now) {
			return errorResult(fmt.Sprintf(
				"meeting must be scheduled in the future (server now UTC %s; startsAt %s UTC). "+
					"Collect visit date, local time, and IANA timezone from the caller—do not guess RFC3339.",
				now.Format(time.RFC3339),
				start.UTC().Format(time.RFC3339),
			)), nil, nil
		}
		duration := in.DurationMinutes
		if duration <= 0 {
			duration = 30
		}
		tz := strings.TrimSpace(in.Timezone)
		if tz == "" {
			tz = "UTC"
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "schedule_health_staff_meeting", map[string]any{
			"session_id":  claims.SessionID,
			"staff_id":    in.StaffID,
			"hospital_id": in.HospitalID,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.ScheduleHealthStaffMeeting(ctx, &schedpb.ScheduleHealthStaffMeetingRequest{
			PatientId:       claims.PatientID,
			StaffId:         strings.TrimSpace(in.StaffID),
			HospitalId:      hospitalID,
			StartsAt:        timestamppb.New(start),
			DurationMinutes: duration,
			Timezone:        tz,
			Title:           strings.TrimSpace(in.Title),
			Channel:         "voice",
			CorrelationId:   correlationID,
			VoiceSessionId:  claims.SessionID,
			SendSms:         in.SendSMS,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "schedule_health_staff_meeting", "error", err.Error(), correlationID, nil)
			msg := grpcErrorMessage(err, "scheduling failed")
			if strings.Contains(msg, "patient has not consented to share data with this hospital") {
				if hint := hospitalConsentMismatchMessage(ctx, db, claims.PatientID, hospitalID, activeHospitals); hint != "" {
					msg = hint
				}
			}
			return errorResult(msg), nil, nil
		}
		body, _ := json.Marshal(map[string]any{
			"meetingID":     resp.GetMeeting().GetId(),
			"status":        resp.GetMeeting().GetStatus(),
			"startsAt":      resp.GetMeeting().GetStartsAt().AsTime().Format(time.RFC3339),
			"correlationID": resp.GetMeeting().GetCorrelationId(),
			"message":       "Meeting request is pending staff approval. LiveKit video visit links will be sent after staff accepts or reschedules.",
		})
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "schedule_health_staff_meeting", "success", "", correlationID, map[string]any{
			"meeting_id": resp.GetMeeting().GetId(),
		})
		return textResult(string(body)), nil, nil
	})
}

func grpcErrorMessage(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	if st, ok := status.FromError(err); ok && st.Message() != "" {
		return st.Message()
	}
	return fallback
}
