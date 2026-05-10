package openai

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
)

// SummarizerClient implements outbound.Summarizer using an OpenAI chat model.
type SummarizerClient struct {
	oai openai.Client
}

func NewSummarizerClient(apiKey string) outbound.Summarizer {
	return &SummarizerClient{
		oai: openai.NewClient(
			option.WithAPIKey(apiKey),
		),
	}
}

func (c *SummarizerClient) Summarize(ctx context.Context, chunks []string, focus string) (string, error) {
	if len(chunks) == 0 {
		return "", domainErrors.ErrSummarizeNoChunksProvided
	}

	// Join chunks into a single context string; in a real impl you might truncate to a token budget.
	var sb strings.Builder
	for _, ch := range chunks {
		if strings.TrimSpace(ch) == "" {
			continue
		}
		sb.WriteString("- ")
		sb.WriteString(ch)
		sb.WriteString("\n")
	}
	if sb.Len() == 0 {
		return "", domainErrors.ErrSummarizeAllChunksEmpty
	}

	focusText := "overall clinical summary"
	if focus != "" && focus != "full" {
		focusText = focus
	}

	prompt := fmt.Sprintf(
		"You are a medical summarization assistant.\n\n"+
			"Focus: %s.\n\n"+
			"Here are extracted record chunks:\n\n%s\n\n"+
			"Produce a concise, clinically useful summary in plain English.",
		focusText,
		sb.String(),
	)

	resp, err := c.oai.Responses.New(ctx, responses.ResponseNewParams{
		Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(prompt)},
		Model: openai.ChatModelGPT4oMini,
	})
	if err != nil {
		return "", domainErrors.ErrSummarizeFailed
	}
	return resp.OutputText(), nil
}
