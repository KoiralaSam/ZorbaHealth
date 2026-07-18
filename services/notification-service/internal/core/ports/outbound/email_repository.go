package outbound

import "context"

type EmailSender interface {
	Send(ctx context.Context, toEmail, toName, subject, plainText, html string) error
	SendWithAttachments(ctx context.Context, toEmail, toName, subject, plainText, html string, attachments []EmailAttachment) error
}

type EmailAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}
