package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/domain/errors"
)

const defaultMailtrapSendURL = "https://send.api.mailtrap.io/api/send"

type MailtrapSender struct {
	client    *http.Client
	apiToken  string
	fromEmail string
	fromName  string
	sendURL   string
}

func NewMailtrapSender(apiToken, fromEmail, fromName, sendURL string) *MailtrapSender {
	if strings.TrimSpace(sendURL) == "" {
		sendURL = defaultMailtrapSendURL
	}

	return &MailtrapSender{
		client:    http.DefaultClient,
		apiToken:  apiToken,
		fromEmail: fromEmail,
		fromName:  fromName,
		sendURL:   sendURL,
	}
}

func (s *MailtrapSender) Send(ctx context.Context, toEmail, toName, subject, plainText, html string) error {
	payload := mailtrapMessage{
		From: mailtrapContact{
			Email: s.fromEmail,
			Name:  s.fromName,
		},
		To: []mailtrapContact{
			{
				Email: toEmail,
				Name:  toName,
			},
		},
		Subject: subject,
		Text:    plainText,
		HTML:    html,
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
	From    mailtrapContact   `json:"from"`
	To      []mailtrapContact `json:"to"`
	Subject string            `json:"subject"`
	Text    string            `json:"text,omitempty"`
	HTML    string            `json:"html,omitempty"`
}

type mailtrapContact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}
