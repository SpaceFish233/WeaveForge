package vectordb

import (
	"context"
	"sync"
)

// FallbackEmbedder wraps a primary and fallback embedder.
// It tries the primary first, and if it fails or isn't ready, uses the fallback.
type FallbackEmbedder struct {
	mu       sync.RWMutex
	primary  Embedder
	fallback Embedder
}

func NewFallbackEmbedder(primary, fallback Embedder) *FallbackEmbedder {
	return &FallbackEmbedder{primary: primary, fallback: fallback}
}

func (e *FallbackEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	e.mu.RLock()
	primary := e.primary
	fallback := e.fallback
	e.mu.RUnlock()

	vec, err := primary.Embed(ctx, text)
	if err != nil {
		return fallback.Embed(ctx, text)
	}
	return vec, nil
}

func (e *FallbackEmbedder) SetPrimary(primary Embedder) {
	e.mu.Lock()
	e.primary = primary
	e.mu.Unlock()
}

func (e *FallbackEmbedder) Engine() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if _, ok := e.primary.(*LlamaCppEmbedder); ok {
		return "llamacpp"
	}
	return "hash"
}

// Ensure interface compliance
var _ Embedder = (*FallbackEmbedder)(nil)
