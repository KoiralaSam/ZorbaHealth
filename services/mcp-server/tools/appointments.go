package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	appointmentpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/appointment"
)

type listAvailableSlotsInput struct {
	StaffID    string `json:"staffID" jsonschema:"hospital staff UUID"`
	HospitalID string `json:"hospitalID" jsonschema:"hospital UUID from list_patient_hospitals"`
	From       string `json:"from,omitempty" jsonschema:"RFC3339 range start; defaults to now"`
	To         string `json:"to,omitempty" jsonschema:"RFC3339 range end; defaults to +14 days"`
	Auth       string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

type getNextAvailableSlotInput struct {
	StaffID    string `json:"staffID" jsonschema:"hospital staff UUID"`
	HospitalID string `json:"hospitalID" jsonschema:"hospital UUID from list_patient_hospitals"`
	After      string `json:"after,omitempty" jsonschema:"RFC3339; defaults to now"`
	Auth       string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

type bookAppointmentInput struct {
	StaffID          string `json:"staffID" jsonschema:"hospital staff UUID"`
	HospitalID       string `json:"hospitalID" jsonschema:"hospital UUID from list_patient_hospitals"`
	StartsAt         string `json:"startsAt" jsonschema:"RFC3339 start time of an available slot"`
	DurationMinutes  int32  `json:"durationMinutes" jsonschema:"appointment length in minutes"`
	Timezone         string `json:"timezone" jsonschema:"IANA timezone"`
	Type             string `json:"type,omitempty" jsonschema:"video or in_person"`
	Title            string `json:"title,omitempty"`
	PatientConfirmed bool   `json:"patientConfirmed" jsonschema:"true only after the caller verbally confirmed hospital, staff, date, and time"`
	SendSMS          bool   `json:"sendSms,omitempty"`
	SendEmail        bool   `json:"sendEmail,omitempty"`
	Auth             string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

type cancelAppointmentInput struct {
	AppointmentID string `json:"appointmentID" jsonschema:"appointment UUID"`
	Reason        string `json:"reason,omitempty"`
	Auth          string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterListAvailableAppointmentSlots(s *mcp.Server, db *pgxpool.Pool, client appointmentpb.AppointmentServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_available_appointment_slots",
		Description: "List bookable appointment slots for a staff member at a consented hospital",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in listAvailableSlotsInput) (*mcp.CallToolResult, any, error) {
		claims, err := requireVerifiedPatient(in.Auth)
		if err != nil {
			return errorResult(err.Error()), nil, nil
		}
		if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.HospitalID) == "" {
			return errorResult("staffID and hospitalID are required"), nil, nil
		}
		hospitalID := strings.TrimSpace(in.HospitalID)
		active, err := queryActivePatientHospitals(ctx, db, claims.PatientID)
		if err != nil {
			return errorResult("failed to verify hospital consent"), nil, nil
		}
		if msg := hospitalConsentMismatchMessage(ctx, db, claims.PatientID, hospitalID, active); msg != "" {
			return errorResult(msg), nil, nil
		}
		from := time.Now().UTC()
		to := from.AddDate(0, 0, 14)
		if strings.TrimSpace(in.From) != "" {
			if parsed, perr := time.Parse(time.RFC3339, strings.TrimSpace(in.From)); perr == nil {
				from = parsed
			}
		}
		if strings.TrimSpace(in.To) != "" {
			if parsed, perr := time.Parse(time.RFC3339, strings.TrimSpace(in.To)); perr == nil {
				to = parsed
			}
		}
		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "list_available_appointment_slots", map[string]any{
			"hospital_id": hospitalID,
			"staff_id":    in.StaffID,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.ListAvailableSlots(ctx, &appointmentpb.ListAvailableSlotsRequest{
			StaffId:    strings.TrimSpace(in.StaffID),
			HospitalId: hospitalID,
			From:       timestamppb.New(from),
			To:         timestamppb.New(to),
			Limit:      20,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "list_available_appointment_slots", "error", err.Error(), correlationID, nil)
			return errorResult(grpcErrorMessage(err, "failed to list slots")), nil, nil
		}
		slots := make([]map[string]any, 0, len(resp.GetSlots()))
		for _, s := range resp.GetSlots() {
			slots = append(slots, map[string]any{
				"starts_at":        s.GetStartsAt().AsTime().Format(time.RFC3339),
				"ends_at":          s.GetEndsAt().AsTime().Format(time.RFC3339),
				"duration_minutes": s.GetDurationMinutes(),
				"timezone":         s.GetTimezone(),
			})
		}
		body, _ := json.Marshal(map[string]any{"slots": slots})
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "list_available_appointment_slots", "success", "", correlationID, map[string]any{"count": len(slots)})
		return textResult(string(body)), nil, nil
	})
}

func RegisterGetNextAvailableSlot(s *mcp.Server, db *pgxpool.Pool, client appointmentpb.AppointmentServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_next_available_slot",
		Description: "Return the earliest bookable appointment slot for staff (auto-select for voice/UI)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in getNextAvailableSlotInput) (*mcp.CallToolResult, any, error) {
		claims, err := requireVerifiedPatient(in.Auth)
		if err != nil {
			return errorResult(err.Error()), nil, nil
		}
		_ = claims
		if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.HospitalID) == "" {
			return errorResult("staffID and hospitalID are required"), nil, nil
		}
		after := time.Now().UTC()
		if strings.TrimSpace(in.After) != "" {
			if parsed, perr := time.Parse(time.RFC3339, strings.TrimSpace(in.After)); perr == nil {
				after = parsed
			}
		}
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.GetNextAvailableSlot(ctx, &appointmentpb.GetNextAvailableSlotRequest{
			StaffId:    strings.TrimSpace(in.StaffID),
			HospitalId: strings.TrimSpace(in.HospitalID),
			After:      timestamppb.New(after),
		})
		if err != nil {
			return errorResult(grpcErrorMessage(err, "no available slot")), nil, nil
		}
		s := resp.GetSlot()
		body, _ := json.Marshal(map[string]any{
			"starts_at":        s.GetStartsAt().AsTime().Format(time.RFC3339),
			"ends_at":          s.GetEndsAt().AsTime().Format(time.RFC3339),
			"duration_minutes": s.GetDurationMinutes(),
			"timezone":         s.GetTimezone(),
		})
		return textResult(string(body)), nil, nil
	})
}

func RegisterBookAppointment(s *mcp.Server, db *pgxpool.Pool, client appointmentpb.AppointmentServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "book_appointment",
		Description: "Book an appointment into an available staff slot after the patient confirms details",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in bookAppointmentInput) (*mcp.CallToolResult, any, error) {
		claims, err := requireVerifiedPatient(in.Auth)
		if err != nil {
			return errorResult(err.Error()), nil, nil
		}
		if !sharedauth.HasScope(claims, "records:read") {
			return errorResult("forbidden: missing records:read"), nil, nil
		}
		if !in.PatientConfirmed {
			return errorResult("patientConfirmed must be true after you verbally confirmed hospital, staff member, date, and time with the caller"), nil, nil
		}
		if strings.TrimSpace(in.StaffID) == "" || strings.TrimSpace(in.HospitalID) == "" {
			return errorResult("staffID and hospitalID are required"), nil, nil
		}
		hospitalID := strings.TrimSpace(in.HospitalID)
		active, err := queryActivePatientHospitals(ctx, db, claims.PatientID)
		if err != nil {
			return errorResult("failed to verify hospital consent"), nil, nil
		}
		if msg := hospitalConsentMismatchMessage(ctx, db, claims.PatientID, hospitalID, active); msg != "" {
			return errorResult(msg), nil, nil
		}
		start, err := time.Parse(time.RFC3339, strings.TrimSpace(in.StartsAt))
		if err != nil {
			return errorResult("startsAt must be RFC3339"), nil, nil
		}
		duration := in.DurationMinutes
		if duration <= 0 {
			duration = 30
		}
		tz := strings.TrimSpace(in.Timezone)
		if tz == "" {
			tz = "UTC"
		}
		typ := strings.TrimSpace(in.Type)
		if typ == "" {
			typ = "video"
		}
		correlationID := auditStart(ctx, claims, sharedaudit.EventAIToolCalled, "book_appointment", map[string]any{
			"hospital_id": hospitalID,
			"staff_id":    in.StaffID,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.BookAppointment(ctx, &appointmentpb.BookAppointmentRequest{
			PatientId:         claims.PatientID,
			StaffId:           strings.TrimSpace(in.StaffID),
			HospitalId:        hospitalID,
			StartsAt:          timestamppb.New(start),
			DurationMinutes:   duration,
			Timezone:          tz,
			Type:              typ,
			Channel:           "voice",
			Title:             strings.TrimSpace(in.Title),
			CorrelationId:     correlationID,
			VoiceSessionId:    claims.SessionID,
			BookedByActorType: "patient",
			BookedByActorId:   claims.PatientID,
			SendSms:           in.SendSMS,
			SendEmail:         in.SendEmail,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "book_appointment", "error", err.Error(), correlationID, nil)
			return errorResult(grpcErrorMessage(err, "booking failed")), nil, nil
		}
		a := resp.GetAppointment()
		body, _ := json.Marshal(map[string]any{
			"appointmentID": a.GetId(),
			"status":        a.GetStatus(),
			"startsAt":      a.GetStartsAt().AsTime().Format(time.RFC3339),
			"type":          a.GetType(),
			"message":       "Appointment booked. Join details are available for video appointments.",
		})
		auditComplete(ctx, db, claims, sharedaudit.EventAIToolCalled, "book_appointment", "success", "", correlationID, map[string]any{
			"appointment_id": a.GetId(),
		})
		return textResult(string(body)), nil, nil
	})
}

func RegisterCancelAppointment(s *mcp.Server, db *pgxpool.Pool, client appointmentpb.AppointmentServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "cancel_appointment",
		Description: "Cancel a previously booked appointment for the verified patient",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in cancelAppointmentInput) (*mcp.CallToolResult, any, error) {
		claims, err := requireVerifiedPatient(in.Auth)
		if err != nil {
			return errorResult(err.Error()), nil, nil
		}
		_ = claims
		if strings.TrimSpace(in.AppointmentID) == "" {
			return errorResult("appointmentID is required"), nil, nil
		}
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.CancelAppointment(ctx, &appointmentpb.CancelAppointmentRequest{
			AppointmentId: strings.TrimSpace(in.AppointmentID),
			Reason:        in.Reason,
		})
		if err != nil {
			return errorResult(grpcErrorMessage(err, "cancel failed")), nil, nil
		}
		body, _ := json.Marshal(map[string]any{
			"appointmentID": resp.GetAppointment().GetId(),
			"status":        resp.GetAppointment().GetStatus(),
		})
		return textResult(string(body)), nil, nil
	})
}

func requireVerifiedPatient(auth string) (*sharedauth.Claims, error) {
	if err := requireToken(auth); err != nil {
		return nil, fmt.Errorf("unauthorized")
	}
	claims, err := verifyToken(auth)
	if err != nil {
		return nil, fmt.Errorf("unauthorized")
	}
	if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
		return nil, err
	}
	if claims.PatientID == "" || strings.HasPrefix(claims.PatientID, "session:") {
		return nil, fmt.Errorf("identity verification is required")
	}
	return claims, nil
}
