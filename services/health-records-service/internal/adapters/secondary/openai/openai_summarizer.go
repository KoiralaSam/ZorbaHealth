package openai

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"

	domainErrors "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/domain/errors"
	sharedproviders "github.com/KoiralaSam/ZorbaHealth/shared/ports/providers"
)

const llmModelName = "gpt-4o-mini"

// SummarizerClient implements outbound.Summarizer using an OpenAI chat model.
type SummarizerClient struct {
	oai openai.Client
}

var _ sharedproviders.LLMProvider = (*SummarizerClient)(nil)

func NewSummarizerClient(apiKey string) *SummarizerClient {
	return &SummarizerClient{
		oai: openai.NewClient(
			option.WithAPIKey(apiKey),
		),
	}
}

func NewLLMProvider(providerName, apiKey string) (sharedproviders.LLMProvider, error) {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "", providerOpenAI:
		return NewSummarizerClient(apiKey), nil
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", providerName)
	}
}

func (c *SummarizerClient) ProviderName() string {
	return providerOpenAI
}

func (c *SummarizerClient) ModelName() string {
	return llmModelName
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
			"Produce a concise, clinically useful summary in plain English. "+
			"If the excerpts do not contain information for the requested focus, say that no matching details were found in the available records.",
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

func (c *SummarizerClient) AnswerQuestion(ctx context.Context, question string, chunks []string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", domainErrors.ErrQueryRequired
	}
	if len(chunks) == 0 {
		return "", domainErrors.ErrSummarizeNoChunksProvided
	}

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

	prompt := fmt.Sprintf(
		"You answer patient questions using only the provided medical record excerpts.\n\n"+
			"Question: %s\n\n"+
			"Record excerpts:\n%s\n\n"+
			"Write a brief, plain-English answer. If the excerpts do not contain the answer, say that you do not see it in the available records.",
		strings.TrimSpace(question),
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
