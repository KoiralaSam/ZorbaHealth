package services

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/domain/calendar"
	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/inbound"
	outbound "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/outbound"
	notificationtemplates "github.com/KoiralaSam/ZorbaHealth/services/notification-service/templates"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/meetingjoin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var notificationTracer = otel.Tracer("notification-service")

type NotificationService struct {
	email         outbound.EmailSender
	sms           outbound.SMSSender
	inbound       inbound.SMSReceiver
	voiceSMS      inbound.VoiceSMSProcessor
	audit         outbound.NotificationAudit
	publicWebBase string
}

func NewNotificationService(
	email outbound.EmailSender,
	sms outbound.SMSSender,
	inbound inbound.SMSReceiver,
	voiceSMS inbound.VoiceSMSProcessor,
	audit outbound.NotificationAudit,
	publicWebBase string,
) *NotificationService {
	return &NotificationService{
		email:         email,
		sms:           sms,
		inbound:       inbound,
		voiceSMS:      voiceSMS,
		audit:         audit,
		publicWebBase: strings.TrimRight(publicWebBase, "/"),
	}
}

func (s *NotificationService) SendPendingVerificationEmail(ctx context.Context, req *events.PendingRegistrationData, token string) error {
	if req == nil {
		return domainErrors.ErrPendingRegistrationRequestNil
	}
	if req.Email == "" {
		return domainErrors.ErrPendingRegistrationEmailEmpty
	}
	if token == "" {
		return domainErrors.ErrVerificationTokenEmpty
	}
	if s.publicWebBase == "" {
		return domainErrors.ErrPublicWebBaseURLNotConfigured
	}

	verificationURL := s.publicWebBase + "/verify-email?token=" + url.QueryEscape(token)

	subject := "Please verify your email address"
	templateData := map[string]string{
		"FullName":        req.FullName,
		"VerificationURL": verificationURL,
	}
	plain, err := notificationtemplates.RenderText("verification_email.txt.tmpl", templateData)
	if err != nil {
		return err
	}
	html, err := notificationtemplates.RenderHTML("verification_email_html.tmpl", templateData)
	if err != nil {
		return err
	}

	displayName := req.FullName
	if displayName == "" {
		displayName = "there"
	}
	return s.email.Send(ctx, req.Email, displayName, subject, plain, html)
}

// SendOTP sends the OTP to the given phone number via SMS.
func (s *NotificationService) SendOTP(ctx context.Context, phone string, otp string) error {
	if phone == "" {
		return domainErrors.ErrPhoneNumberEmpty
	}
	if otp == "" {
		return domainErrors.ErrOTPEmpty
	}
	message, err := notificationtemplates.RenderText("otp_sms.txt.tmpl", map[string]string{"OTP": otp})
	if err != nil {
		return err
	}
	return s.sms.SendSMS(ctx, phone, message)
}

func (s *NotificationService) SendEmergencyEscalationSMS(ctx context.Context, phone string, reason string) error {
	if phone == "" {
		return domainErrors.ErrPhoneNumberEmpty
	}
	if reason == "" {
		reason = "urgent symptoms"
	}
	message, err := notificationtemplates.RenderText("emergency_escalation_notice.txt.tmpl", map[string]string{"Reason": reason})
	if err != nil {
		return err
	}
	return s.sms.SendSMS(ctx, phone, message)
}

func (s *NotificationService) SendEmergencyEscalationAlerts(ctx context.Context, phones []string, reason string) []error {
	if len(phones) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(phones))
	var errs []error
	for _, phone := range phones {
		normalized := strings.TrimSpace(phone)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}

		if err := s.SendEmergencyEscalationSMS(ctx, normalized, reason); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// ReceiveSMS forwards inbound SMS messages (webhook) to the configured inbound receiver.
func (s *NotificationService) ReceiveSMS(ctx context.Context, phoneNumber, message string) error {
	if s.inbound != nil {
		if err := s.inbound.ReceiveSMS(ctx, phoneNumber, message); err != nil {
			return err
		}
	}
	if s.voiceSMS != nil {
		_, _, err := s.voiceSMS.ProcessInboundVoiceSms(ctx, phoneNumber, message)
		return err
	}
	return nil
}

func (s *NotificationService) SendMeetingRequestedNotifications(ctx context.Context, data *events.MeetingRequestedData) error {
	if data == nil || data.StaffEmail == "" {
		return nil
	}
	ctx, span := notificationTracer.Start(ctx, "notification.meeting_requested")
	defer span.End()
	span.SetAttributes(attribute.String("meeting.id", data.MeetingID))

	start, err := time.Parse(time.RFC3339, data.StartsAtRFC3339)
	if err != nil {
		span.RecordError(err)
		return err
	}
	subject := fmt.Sprintf("Visit request pending approval: %s", data.Title)
	templateData := map[string]string{
		"StaffName":    data.StaffName,
		"PatientName":  data.PatientName,
		"StartRFC1123": start.Format(time.RFC1123),
	}
	plain, err := notificationtemplates.RenderText("meeting_requested_plain.tmpl", templateData)
	if err != nil {
		return err
	}
	html, err := notificationtemplates.RenderHTML("meeting_requested_html.tmpl", templateData)
	if err != nil {
		return err
	}
	if err := s.email.Send(ctx, data.StaffEmail, data.StaffName, subject, plain, html); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (s *NotificationService) SendMeetingScheduledNotifications(ctx context.Context, data *events.MeetingScheduledData) error {
	if data == nil {
		return nil
	}
	ctx, span := notificationTracer.Start(ctx, "notification.meeting_scheduled")
	defer span.End()
	span.SetAttributes(attribute.String("meeting.id", data.MeetingID))

	start, err := time.Parse(time.RFC3339, data.StartsAtRFC3339)
	if err != nil {
		span.RecordError(err)
		return err
	}

	patientSubject := fmt.Sprintf("Your visit is scheduled: %s", data.Title)
	staffSubject := fmt.Sprintf("Patient visit scheduled: %s", data.Title)
	desc := fmt.Sprintf("Join your Zorba Health video visit: %s", data.JoinURL)

	patientJoinURL := meetingJoinURLForNotification(s.publicWebBase, data.JoinURL, data.LiveKitRoomName, data.PatientToken, "patient")
	staffJoinURL := meetingJoinURLForNotification(s.publicWebBase, data.JoinURL, data.LiveKitRoomName, data.StaffToken, "staff")

	if data.PatientEmail != "" {
		ics := calendar.BuildMeetingRequestICS(
			calendar.MeetingUID(data.MeetingID, data.PatientEmail),
			data.Title,
			desc,
			s.fromEmail(),
			data.PatientEmail,
			start,
			data.DurationMinutes,
			data.Timezone,
		)
		patientData := map[string]string{
			"PatientName":  data.PatientName,
			"StaffName":    data.StaffName,
			"StartRFC1123": start.Format(time.RFC1123),
			"JoinURL":      patientJoinURL,
		}
		patientPlain, err := notificationtemplates.RenderText("appointment_reminder_patient_plain.tmpl", patientData)
		if err != nil {
			return err
		}
		patientHTML, err := notificationtemplates.RenderHTML("appointment_reminder_patient_html.tmpl", patientData)
		if err != nil {
			return err
		}
		err = s.email.SendWithAttachments(ctx, data.PatientEmail, data.PatientName, patientSubject, patientPlain, patientHTML, []outbound.EmailAttachment{{
			Filename:    "appointment.ics",
			ContentType: "text/calendar; method=REQUEST",
			Content:     []byte(ics),
		}})
		s.recordNotifyAudit(ctx, data, "email", "patient", err)
		if err != nil {
			span.RecordError(err)
		}
	}

	if data.StaffEmail != "" {
		ics := calendar.BuildMeetingRequestICS(
			calendar.MeetingUID(data.MeetingID, data.StaffEmail),
			data.Title,
			desc,
			s.fromEmail(),
			data.StaffEmail,
			start,
			data.DurationMinutes,
			data.Timezone,
		)
		staffData := map[string]string{
			"StaffName":   data.StaffName,
			"PatientName": data.PatientName,
			"JoinURL":     staffJoinURL,
		}
		staffPlain, err := notificationtemplates.RenderText("appointment_reminder_staff_plain.tmpl", staffData)
		if err != nil {
			return err
		}
		staffHTML, err := notificationtemplates.RenderHTML("appointment_reminder_staff_html.tmpl", staffData)
		if err != nil {
			return err
		}
		err = s.email.SendWithAttachments(ctx, data.StaffEmail, data.StaffName, staffSubject, staffPlain, staffHTML, []outbound.EmailAttachment{{
			Filename:    "appointment.ics",
			ContentType: "text/calendar; method=REQUEST",
			Content:     []byte(ics),
		}})
		s.recordNotifyAudit(ctx, data, "email", "staff", err)
		if err != nil {
			span.RecordError(err)
		}
	}

	if data.SendSMS && data.PatientPhone != "" {
		smsMsg := fmt.Sprintf("Zorba Health: video visit with %s on %s. Join: %s",
			data.StaffName, start.Format("Jan 2 3:04 PM MST"), data.JoinURL)
		err := s.sms.SendSMS(ctx, data.PatientPhone, smsMsg)
		s.recordNotifyAudit(ctx, data, "sms", "patient", err)
		if err != nil {
			span.RecordError(err)
		}
	}
	return nil
}

func meetingJoinURLForNotification(webBase, storedJoinURL, roomName, token, role string) string {
	storedJoinURL = strings.TrimSpace(storedJoinURL)
	if storedJoinURL == "" {
		return ""
	}
	roomName = strings.TrimSpace(roomName)
	if roomName == "" {
		roomName = meetingjoin.RoomFromJoinURL(storedJoinURL)
	}
	serverWS := meetingjoin.LiveKitServerURL(storedJoinURL)
	if web := meetingjoin.WebAppJoinURL(webBase, serverWS, roomName, token); web != "" {
		return web
	}
	return meetingjoin.WithParticipantToken(storedJoinURL, token, role)
}

func (s *NotificationService) fromEmail() string {
	return "care@zorba.health"
}

func (s *NotificationService) recordNotifyAudit(ctx context.Context, data *events.MeetingScheduledData, channel, role string, err error) {
	if s.audit == nil {
		return
	}
	fail := ""
	success := err == nil
	if err != nil {
		fail = err.Error()
	}
	_ = s.audit.RecordNotificationSent(ctx, data.PatientID, data.CorrelationID, data.MeetingID, channel, role, "meeting_scheduled", success, fail)
}
