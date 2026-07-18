package voipms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/notification-service/internal/core/domain/errors"
	sharedproviders "github.com/KoiralaSam/ZorbaHealth/shared/ports/providers"
)

// Sender sends SMS via VoIP.ms REST/JSON API.
// VoIP.ms expects: api_username, api_password, method and required parameters in the URL; &content_type=json for JSON response.
// See https://www.voip.ms/api.php
type Sender struct {
	client   *http.Client
	baseURL  string // e.g. https://voip.ms/api/v1/rest.php (no query string)
	username string // api_username (account email)
	password string // api_password (API password from account settings)
	did      string // from number (DID)
}

var _ sharedproviders.SMSProvider = (*Sender)(nil)

// NewSender creates a VoIP.ms SMS sender. baseURL is the REST endpoint (e.g. rest.php).
// username and password are the VoIP.ms API credentials (api_username, api_password) sent in the URL.
func NewSender(baseURL, username, password, did string) *Sender {
	if baseURL == "" {
		baseURL = "https://voip.ms/api/v1/rest.php"
	}
	return &Sender{
		client:   &http.Client{},
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		username: username,
		password: password,
		did:      did,
	}
}

// SendSMS implements outbound.SMSSender using VoIP.ms REST API.
// Method and parameters are sent in the URL; content_type=json requests JSON response.
func (s *Sender) SendSMS(ctx context.Context, toPhoneNumber, message string) error {
	if s.did == "" {
		return domainErrors.ErrVoipmsDIDNotSet
	}
	if toPhoneNumber == "" {
		return domainErrors.ErrVoipmsToPhoneNumberEmpty
	}
	if message == "" {
		return domainErrors.ErrVoipmsMessageEmpty
	}
	if s.username == "" || s.password == "" {
		return domainErrors.ErrVoipmsAPIUsernameRequired
	}

	to := normalizePhone(toPhoneNumber)
	if len(to) != 10 {
		return fmt.Errorf("voipms: destination must be 10-digit NANP after normalization, got %d digits", len(to))
	}
	fromDID := normalizePhone(s.did)

	// VoIP.ms REST/JSON API: api_username, api_password, method and params in the URL; content_type=json for JSON output.
	params := url.Values{}
	params.Set("api_username", s.username)
	params.Set("api_password", s.password)
	params.Set("method", "sendSMS")
	params.Set("did", fromDID)
	// VoIP.ms expects the destination under `dst` for sendSMS.
	params.Set("dst", to)
	params.Set("message", message)
	params.Set("content_type", "json")

	reqURL := s.baseURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("voipms: new request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("voipms: do request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("voipms: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("voipms: http status=%d body=%s", resp.StatusCode, truncateForError(bodyBytes, 512))
	}

	// VoIP.ms commonly returns HTTP 200 even when the request failed; the JSON body contains status=success|error.
	var out struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		if len(bodyBytes) > 0 {
			return fmt.Errorf("voipms: unexpected response: %s", truncateForError(bodyBytes, 512))
		}
		return fmt.Errorf("voipms: empty response body")
	}
	if strings.ToLower(out.Status) != "success" {
		msg := out.Message
		if msg == "" {
			msg = out.Error
		}
		if msg == "" {
			msg = truncateForError(bodyBytes, 512)
		}
		return fmt.Errorf("voipms: api status=%s msg=%s", out.Status, msg)
	}
	return nil
}

func (s *Sender) ProviderName() string {
	return "voipms"
}

func (s *Sender) SendText(ctx context.Context, msg sharedproviders.SMSMessage) error {
	return s.SendSMS(ctx, msg.ToPhoneNumber, msg.Body)
}

func truncateForError(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func normalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	// VoIP.ms `sendSMS` expects a 10-digit NANP destination for US/CA.
	// If we have 11 digits starting with country code 1, strip it.
	if len(digits) == 11 && strings.HasPrefix(digits, "1") {
		return digits[1:]
	}
	return digits
}
