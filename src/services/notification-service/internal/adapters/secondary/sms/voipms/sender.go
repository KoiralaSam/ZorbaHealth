package voipms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
		return fmt.Errorf("voipms: VOIPMS_DID is not set")
	}
	if toPhoneNumber == "" {
		return fmt.Errorf("voipms: to phone number is empty")
	}
	if message == "" {
		return fmt.Errorf("voipms: message is empty")
	}
	if s.username == "" || s.password == "" {
		return fmt.Errorf("voipms: api_username and api_password are required")
	}

	to := normalizePhone(toPhoneNumber)
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

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("voipms: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	// VoIP.ms commonly returns HTTP 200 even when the request failed; the JSON body contains status=success|error.
	var out struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		// If response isn't JSON for some reason, treat non-empty body as a clue.
		if len(bodyBytes) > 0 {
			return fmt.Errorf("voipms: unexpected response: %s", string(bodyBytes))
		}
		return nil
	}
	if strings.ToLower(out.Status) != "success" {
		msg := out.Message
		if msg == "" {
			msg = out.Error
		}
		if msg == "" {
			msg = string(bodyBytes)
		}
		return fmt.Errorf("voipms: api status=%s msg=%s", out.Status, msg)
	}
	return nil
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
