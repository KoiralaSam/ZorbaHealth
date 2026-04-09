package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	sharedenv "github.com/KoiralaSam/ZorbaHealth/shared/env"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
)

type ElevenLabsTTS struct {
	apiKey       string
	baseURL      string
	voiceID      string
	modelID      string
	outputFormat string
	httpClient   *http.Client
}

var _ outbound.Synthesizer = (*ElevenLabsTTS)(nil)

func NewElevenLabs(apiKey string) *ElevenLabsTTS {
	return &ElevenLabsTTS{
		apiKey:       apiKey,
		baseURL:      sharedenv.GetString("ELEVENLABS_BASE_URL", "https://api.elevenlabs.io"),
		voiceID:      sharedenv.GetString("ELEVENLABS_VOICE_ID", ""),
		modelID:      sharedenv.GetString("ELEVENLABS_MODEL_ID", "eleven_flash_v2_5"),
		outputFormat: sharedenv.GetString("ELEVENLABS_OUTPUT_FORMAT", "pcm_24000"),
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

type elevenLabsTTSRequest struct {
	Text    string `json:"text"`
	ModelID string `json:"model_id"`
}

func (e *ElevenLabsTTS) Speak(text string, language string) ([]byte, error) {
	if strings.TrimSpace(e.apiKey) == "" {
		return nil, errors.New("ELEVENLABS_API_KEY is not set")
	}
	if strings.TrimSpace(e.voiceID) == "" {
		return nil, errors.New("ELEVENLABS_VOICE_ID is not set")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("text is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	payload := elevenLabsTTSRequest{
		Text:    text,
		ModelID: e.modelID,
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal elevenlabs request: %w", err)
	}

	url := fmt.Sprintf(
		"%s/v1/text-to-speech/%s?output_format=%s",
		strings.TrimRight(e.baseURL, "/"),
		e.voiceID,
		e.outputFormat,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("create elevenlabs request: %w", err)
	}

	req.Header.Set("xi-api-key", e.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if strings.HasPrefix(e.outputFormat, "pcm_") {
		req.Header.Set("Accept", "audio/pcm")
	} else {
		req.Header.Set("Accept", "audio/mpeg")
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs request failed: %w", err)
	}
	defer resp.Body.Close()

	audio, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read elevenlabs response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elevenlabs returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(audio)))
	}
	if len(audio) == 0 {
		return nil, errors.New("elevenlabs returned empty audio")
	}

	return audio, nil
}
