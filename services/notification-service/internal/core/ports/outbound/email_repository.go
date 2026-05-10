package outbound

import "context"

type EmailSender interface {
	Send(ctx context.Context, toEmail, toName, subject, plainText, html string) error
}
