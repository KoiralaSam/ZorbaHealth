package openai

import (
	"context"
	"fmt"

	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/ports/outbound"
	openai "github.com/openai/openai-go/v3"
	option "github.com/openai/openai-go/v3/option"
)

// Client wraps the official OpenAI Go SDK for text-embedding-3-small.
// Output: 1536 dimensions — matches vector(1536) in Postgres.
//
// CRITICAL RULE: This is the ONLY embedding model used across the entire system.
// health-record-service uses it for ingestion AND search.
// Never swap models without re-embedding every row in record_chunks and conversation_turns.
type Client struct {
	oai openai.Client
}

func NewClient(apiKey string) outbound.Embedder {
	return &Client{
		oai: openai.NewClient(
			option.WithAPIKey(apiKey),
		),
	}
}

func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("embed: empty input")
	}

	resp, err := c.oai.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModelTextEmbedding3Small,
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(text),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embed: empty response")
	}

	// The official SDK returns []float64 — convert to float32 for pgvector.
	// This is lossless at the precision level embeddings operate at.
	f64 := resp.Data[0].Embedding
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32, nil
}
