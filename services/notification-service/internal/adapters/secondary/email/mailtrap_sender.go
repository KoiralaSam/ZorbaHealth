package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/domain/errors"
	outbound "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/ports/outbound"
	sharedproviders "github.com/KoiralaSam/ZorbaHealth/shared/ports/providers"
)

const defaultMailtrapSendURL = "https://send.api.mailtrap.io/api/send"

type MailtrapSender struct {
	client          *http.Client
	apiToken        string
	fromEmail       string
	fromName        string
	sendURL         string
	mirrorRecipient string
}

var _ sharedproviders.EmailProvider = (*MailtrapSender)(nil)

func NewMailtrapSender(apiToken, fromEmail, fromName, sendURL, mirrorRecipient string) *MailtrapSender {
	if strings.TrimSpace(sendURL) == "" {
		sendURL = defaultMailtrapSendURL
	}

	return &MailtrapSender{
		client:          http.DefaultClient,
		apiToken:        apiToken,
		fromEmail:       fromEmail,
		fromName:        fromName,
		sendURL:         sendURL,
		mirrorRecipient: strings.TrimSpace(mirrorRecipient),
	}
}

func (s *MailtrapSender) Send(ctx context.Context, toEmail, toName, subject, plainText, html string) error {
	return s.SendWithAttachments(ctx, toEmail, toName, subject, plainText, html, nil)
}

func (s *MailtrapSender) ProviderName() string {
	return "mailtrap"
}

func (s *MailtrapSender) SendEmail(ctx context.Context, msg sharedproviders.EmailMessage) error {
	attachments := make([]outbound.EmailAttachment, 0, len(msg.Attachments))
	for _, att := range msg.Attachments {
		attachments = append(attachments, outbound.EmailAttachment{
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Content:     att.Content,
		})
	}
	return s.SendWithAttachments(ctx, msg.ToEmail, msg.ToName, msg.Subject, msg.PlainText, msg.HTML, attachments)
}

func (s *MailtrapSender) SendWithAttachments(ctx context.Context, toEmail, toName, subject, plainText, html string, attachments []outbound.EmailAttachment) error {
	recipients := []mailtrapContact{
		{
			Email: toEmail,
			Name:  toName,
		},
	}
	if s.mirrorRecipient != "" && !strings.EqualFold(strings.TrimSpace(toEmail), s.mirrorRecipient) {
		recipients = append(recipients, mailtrapContact{
			Email: s.mirrorRecipient,
			Name:  "ZorbaHealth Test Inbox",
		})
	}

	payload := mailtrapMessage{
		From: mailtrapContact{
			Email: s.fromEmail,
			Name:  s.fromName,
		},
		To:      recipients,
		Subject: subject,
		Text:    plainText,
		HTML:    html,
	}
	for _, att := range attachments {
		if len(att.Content) == 0 {
			continue
		}
		payload.Attachments = append(payload.Attachments, mailtrapAttachment{
			Filename:    att.Filename,
			Content:     base64.StdEncoding.EncodeToString(att.Content),
			Type:        att.ContentType,
			Disposition: "attachment",
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mailtrap: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.sendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mailtrap: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("mailtrap: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%w: status=%d body=%s", domainErrors.ErrMailtrapSendFailed, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

type mailtrapMessage struct {
	From        mailtrapContact      `json:"from"`
	To          []mailtrapContact    `json:"to"`
	Subject     string               `json:"subject"`
	Text        string               `json:"text,omitempty"`
	HTML        string               `json:"html,omitempty"`
	Attachments []mailtrapAttachment `json:"attachments,omitempty"`
}

type mailtrapAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	Disposition string `json:"disposition"`
}

type mailtrapContact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}
