package services

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	outbound "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
)

type NotificationService struct {
	email         outbound.EmailSender
	publicWebBase string
}

func NewNotificationService(email outbound.EmailSender, publicWebBase string) *NotificationService {
	return &NotificationService{
		email:         email,
		publicWebBase: strings.TrimRight(publicWebBase, "/"),
	}
}

func (s *NotificationService) SendPendingVerificationEmail(ctx context.Context, req *events.PendingRegistrationData, token string) error {
	if req == nil {
		return fmt.Errorf("pending registration request is nil")
	}
	if req.Email == "" {
		return fmt.Errorf("pending registration email is empty")
	}
	if token == "" {
		return fmt.Errorf("verification token is empty")
	}
	if s.publicWebBase == "" {
		return fmt.Errorf("PUBLIC_WEB_BASE_URL is not configured")
	}

	verificationURL := s.publicWebBase + "/verify-email?token=" + url.QueryEscape(token)

	subject := "Please verify your email address"

	plain := fmt.Sprintf(
		"Hey %s,\n\n"+
			"To complete your Zorba Health registration, please click the button below (or copy and paste the link into your browser) to confirm this is your correct email address:\n\n%s\n\n"+
			"This verification link will expire in 24 hours. If you didn’t request this, you can safely ignore this email.\n",
		req.FullName,
		verificationURL,
	)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Email verification</title>
  </head>
  <body style="margin:0;padding:0;background-color:#f5f7fb;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
    <table role="presentation" cellpadding="0" cellspacing="0" width="100%%" style="background-color:#f5f7fb;padding:24px 0;">
      <tr>
        <td align="center">
          <table role="presentation" cellpadding="0" cellspacing="0" width="560" style="background-color:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 4px 20px rgba(15,23,42,0.08);">
            <tr>
              <td align="center" style="padding:24px 24px 8px 24px;background:linear-gradient(135deg,#12b981,#059669);color:#ffffff;">
                <div style="font-size:24px;font-weight:700;letter-spacing:0.04em;text-transform:uppercase;">
                  Your email<br />address needs verifying
                </div>
              </td>
            </tr>
            <tr>
              <td style="padding:24px 32px 28px 32px;color:#111827;font-size:16px;line-height:1.5;">
                <p style="margin:0 0 16px 0;">Hey %s,</p>
                <p style="margin:0 0 16px 0;">
                  To complete the email verification process, please click the button below to
                  confirm that this is your correct email address.
                </p>
                <p style="margin:24px 0;" align="center">
                  <a href="%s"
                     style="display:inline-block;background-color:#f97316;color:#ffffff;text-decoration:none;padding:14px 32px;border-radius:999px;font-weight:600;font-size:15px;">
                    Verify your email
                  </a>
                </p>
                <p style="margin:0 0 12px 0;font-size:13px;color:#6b7280;">
                  This verification link will expire in 24 hours. If you didn’t make a change to
                  your account, don’t worry – this update won’t happen unless you verify your email.
                </p>
                <p style="margin:16px 0 0 0;font-size:13px;color:#6b7280;">
                  If the button above doesn’t work, copy and paste this link into your browser:<br />
                  <span style="word-break:break-all;color:#059669;">%s</span>
                </p>
                <p style="margin:24px 0 0 0;font-size:13px;color:#9ca3af;">
                  Happy verifying,<br />
                  <strong>Zorba Health</strong>
                </p>
              </td>
            </tr>
          </table>
        </td>
      </tr>
    </table>
  </body>
</html>`, req.FullName, verificationURL, verificationURL)

	displayName := req.FullName
	if displayName == "" {
		displayName = "there"
	}
	return s.email.Send(ctx, req.Email, displayName, subject, plain, html)
}
