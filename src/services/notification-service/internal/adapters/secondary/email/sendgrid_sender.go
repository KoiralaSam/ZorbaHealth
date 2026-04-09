package email

import (
	"context"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/domain/errors"
)

type SendGridSender struct {
	client    *sendgrid.Client
	fromEmail string
	fromName  string
}

func NewSendGridSender(apiKey, fromEmail, fromName string) *SendGridSender {
	return &SendGridSender{
		client:    sendgrid.NewSendClient(apiKey),
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (s *SendGridSender) Send(ctx context.Context, toEmail, toName, subject, plainText, html string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(toName, toEmail)
	msg := mail.NewSingleEmail(from, subject, to, plainText, html)

	resp, err := s.client.SendWithContext(ctx, msg)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domainErrors.ErrSendGridSendFailed
	}
	return nil
}

