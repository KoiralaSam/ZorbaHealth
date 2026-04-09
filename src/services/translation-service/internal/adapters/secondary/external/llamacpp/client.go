package llamacpp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/models"
)

const (
	chatCompletionsPath = "/v1/chat/completions"
	maxResponseBodySize = 1 << 20
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

func NewClient(baseURL string, timeout time.Duration, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type chatCompletionRequest struct {
	Model       string    `json:"model,omitempty"`
	Temperature float64   `json:"temperature"`
	TopP        float64   `json:"top_p"`
	Messages    []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

func (c *Client) Translate(ctx context.Context, req models.TranslationRequest) (*models.TranslationResult, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("%w: TRANSLATION_MODEL_BASE_URL not configured", domainerrors.ErrProviderUnavailable)
	}

	payload := chatCompletionRequest{
		Model:       c.model,
		Temperature: 0,
		TopP:        1,
		Messages: []message{
			{
				Role: "system",
				Content: "You are a translation engine. Translate the user text exactly into the requested target language. " +
					"Do not explain. Do not summarize. Do not add notes. Return only the translated text.",
			},
			{
				Role:    "user",
				Content: buildPrompt(req),
			},
		},
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+chatCompletionsPath, bytes.NewReader(rawBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "ZorbaHealth/translation-service")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domainerrors.ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxResponseBodySize)
	rawResp, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: model server returned status %d", domainerrors.ErrProviderUnavailable, resp.StatusCode)
	}

	var decoded chatCompletionResponse
	if err := json.Unmarshal(rawResp, &decoded); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("%w: empty choices", domainerrors.ErrTranslationFailed)
	}

	translated := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if translated == "" {
		return nil, fmt.Errorf("%w: empty translated text", domainerrors.ErrTranslationFailed)
	}

	detectedLang := ""
	if req.SourceLang != "" {
		detectedLang = strings.ToLower(req.SourceLang)
	}

	return &models.TranslationResult{
		TranslatedText: translated,
		DetectedLang:   detectedLang,
		CharacterCount: utf8.RuneCountInString(req.Text),
	}, nil
}

func buildPrompt(req models.TranslationRequest) string {
	if req.SourceLang == "" {
		return fmt.Sprintf("Translate the following text to %s:\n\n%s", req.TargetLang, req.Text)
	}
	return fmt.Sprintf("Translate the following text from %s to %s:\n\n%s", req.SourceLang, req.TargetLang, req.Text)
}
