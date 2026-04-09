package deepgram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
)

type DeepgramSTT struct {
	apiKey      string
	baseURL     string
	model       string
	contentType string
	encoding    string
	sampleRate  string
	httpClient  *http.Client
}

var _ outbound.Transcriber = (*DeepgramSTT)(nil)

func NewDeepgram(apiKey string) *DeepgramSTT {
	return &DeepgramSTT{
		apiKey:      apiKey,
		baseURL:     sharedenv.GetString("DEEPGRAM_BASE_URL", "https://api.deepgram.com"),
		model:       sharedenv.GetString("DEEPGRAM_MODEL", "nova-3"),
		contentType: sharedenv.GetString("DEEPGRAM_AUDIO_CONTENT_TYPE", "audio/wav"),
		encoding:    sharedenv.GetString("DEEPGRAM_ENCODING", ""),
		sampleRate:  sharedenv.GetString("DEEPGRAM_SAMPLE_RATE", ""),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type deepgramListenResponse struct {
	Results struct {
		Channels []struct {
			Alternatives []struct {
				Transcript string `json:"transcript"`
			} `json:"alternatives"`
		} `json:"channels"`
	} `json:"results"`
	ErrMsg string `json:"err_msg,omitempty"`
}

func (d *DeepgramSTT) Transcribe(audio []byte, language string) (string, error) {
	if strings.TrimSpace(d.apiKey) == "" {
		return "", errors.New("DEEPGRAM_API_KEY is not set")
	}
	if len(audio) == 0 {
		return "", errors.New("audio is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	u, err := url.Parse(strings.TrimRight(d.baseURL, "/") + "/v1/listen")
	if err != nil {
		return "", fmt.Errorf("parse deepgram url: %w", err)
	}

	q := u.Query()
	q.Set("model", d.model)
	q.Set("smart_format", "true")
	if strings.TrimSpace(language) != "" && !strings.EqualFold(language, "auto") {
		q.Set("language", language)
	}
	if strings.TrimSpace(d.encoding) != "" {
		q.Set("encoding", d.encoding)
	}
	if strings.TrimSpace(d.sampleRate) != "" {
		q.Set("sample_rate", d.sampleRate)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(audio))
	if err != nil {
		return "", fmt.Errorf("create deepgram request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+d.apiKey)
	req.Header.Set("Content-Type", d.contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepgram request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read deepgram response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("deepgram returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed deepgramListenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse deepgram response: %w", err)
	}

	if len(parsed.Results.Channels) == 0 || len(parsed.Results.Channels[0].Alternatives) == 0 {
		return "", errors.New("deepgram returned no transcript alternatives")
	}

	transcript := strings.TrimSpace(parsed.Results.Channels[0].Alternatives[0].Transcript)
	if transcript == "" {
		return "", errors.New("deepgram returned empty transcript")
	}

	return transcript, nil
}
