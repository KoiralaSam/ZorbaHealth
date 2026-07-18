package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
)

func TestMeetingRequestedNotificationHasNoJoinURL(t *testing.T) {
	email := &fakeEmailSender{}
	svc := NewNotificationService(email, &fakeSMSSender{}, nil, nil, nil, "http://localhost:3000")
	err := svc.SendMeetingRequestedNotifications(context.Background(), &events.MeetingRequestedData{
		MeetingID:       "meeting-1",
		PatientID:       "patient-1",
		StaffID:         "staff-1",
		HospitalID:      "hospital-1",
		CorrelationID:   "corr-1",
		StartsAtRFC3339: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		DurationMinutes: 30,
		Timezone:        "UTC",
		Title:           "Visit",
		PatientName:     "Pat Patient",
		StaffEmail:      "staff@example.com",
		StaffName:       "Dr Staff",
	})
	if err != nil {
		t.Fatal(err)
	}
	if email.lastTo != "staff@example.com" {
		t.Fatalf("email to = %q", email.lastTo)
	}
	if strings.Contains(email.lastPlain, "token=") || strings.Contains(email.lastHTML, "token=") {
		t.Fatalf("pending notification leaked join token: %s", email.lastPlain)
	}
}

func TestMeetingScheduledNotificationIncludesPatientJoinURL(t *testing.T) {
	email := &fakeEmailSender{}
	audit := &fakeNotificationAudit{}
	svc := NewNotificationService(email, &fakeSMSSender{}, nil, nil, audit, "http://localhost:3000")
	err := svc.SendMeetingScheduledNotifications(context.Background(), &events.MeetingScheduledData{
		MeetingID:       "meeting-1",
		PatientID:       "patient-1",
		StaffID:         "staff-1",
		HospitalID:      "hospital-1",
		CorrelationID:   "corr-1",
		StartsAtRFC3339: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		DurationMinutes: 30,
		Timezone:        "UTC",
		Title:           "Visit",
		JoinURL:         "http://localhost:3000/meeting/join?server=wss%3A%2F%2Flivekit.example&room=meeting-1&token=token-1",
		PatientEmail:    "patient@example.com",
		PatientName:     "Pat Patient",
		StaffEmail:      "staff@example.com",
		StaffName:       "Dr Staff",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(email.sent) != 2 {
		t.Fatalf("emails sent = %d, want 2", len(email.sent))
	}
	patient := email.sent[0]
	if !strings.Contains(patient.plain, "/meeting/join") {
		t.Fatalf("patient email missing join URL: %s", patient.plain)
	}
	staff := email.sent[1]
	if !strings.Contains(staff.plain, "/meeting/join") {
		t.Fatalf("staff email missing join URL: %s", staff.plain)
	}
}

type sentEmail struct {
	to    string
	plain string
	html  string
}

type fakeEmailSender struct {
	lastTo    string
	lastPlain string
	lastHTML  string
	sent      []sentEmail
}

func (f *fakeEmailSender) Send(ctx context.Context, toEmail, toName, subject, plainText, html string) error {
	f.lastTo = toEmail
	f.lastPlain = plainText
	f.lastHTML = html
	f.sent = append(f.sent, sentEmail{to: toEmail, plain: plainText, html: html})
	return nil
}

func (f *fakeEmailSender) SendWithAttachments(ctx context.Context, toEmail, toName, subject, plainText, html string, attachments []outbound.EmailAttachment) error {
	return f.Send(ctx, toEmail, toName, subject, plainText, html)
}

type fakeSMSSender struct{}

func (f *fakeSMSSender) SendSMS(ctx context.Context, toPhoneNumber, message string) error {
	return nil
}

type fakeNotificationAudit struct{}

func (f *fakeNotificationAudit) RecordNotificationSent(ctx context.Context, patientID, correlationID, meetingID, channel, recipientRole, template string, success bool, failureReason string) error {
	return nil
}
