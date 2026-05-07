package vectordb

import (
	"context"
	"fmt"
)

// LLMEmbedder wraps an LLM client's GetEmbedding as a vectordb.Embedder.
type LLMEmbedder struct {
	client interface {
		GetEmbedding(ctx context.Context, text string, model string) ([]float32, error)
	}
	model string
}

// NewLLMEmbedder creates an Embedder from an LLM-compatible client.
func NewLLMEmbedder(client interface {
	GetEmbedding(ctx context.Context, text string, model string) ([]float32, error)
}, model string) *LLMEmbedder {
	return &LLMEmbedder{client: client, model: model}
}

func (e *LLMEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	emb, err := e.client.GetEmbedding(ctx, text, e.model)
	if err != nil {
		return nil, fmt.Errorf("vectordb: llm embed: %w", err)
	}
	return emb, nil
}
