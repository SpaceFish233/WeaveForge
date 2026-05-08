package vectordb

import "context"

// FallbackEmbedder wraps a primary and fallback embedder.
// It tries the primary first, and if it fails or isn't ready, uses the fallback.
type FallbackEmbedder struct {
	primary  Embedder
	fallback Embedder
}

func NewFallbackEmbedder(primary, fallback Embedder) *FallbackEmbedder {
	return &FallbackEmbedder{primary: primary, fallback: fallback}
}

func (e *FallbackEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vec, err := e.primary.Embed(ctx, text)
	if err != nil {
		return e.fallback.Embed(ctx, text)
	}
	return vec, nil
}

func (e *FallbackEmbedder) SetPrimary(primary Embedder) {
	e.primary = primary
}

func (e *FallbackEmbedder) Engine() string {
	if _, ok := e.primary.(*LlamaCppEmbedder); ok {
		return "llamacpp"
	}
	return "hash"
}

// Ensure interface compliance
var _ Embedder = (*FallbackEmbedder)(nil)
