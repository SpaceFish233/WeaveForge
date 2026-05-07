package vectordb

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// LocalEmbedder generates embeddings from text using character-level n-gram
// hashing.  Zero external dependencies, works fully offline.  Accuracy is
// lower than transformer models, but sufficient for similarity search.
type LocalEmbedder struct {
	dimension int
}

// NewLocalEmbedder creates a local embedder. dimension defaults to 512.
func NewLocalEmbedder(dimension int) *LocalEmbedder {
	if dimension <= 0 {
		dimension = 512
	}
	return &LocalEmbedder{dimension: dimension}
}

// Embed computes a hash-based feature vector from the input text.
func (e *LocalEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if len(text) == 0 {
		return make([]float32, e.dimension), nil
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return make([]float32, e.dimension), nil
	}

	vec := make([]float64, e.dimension)

	// Character unigram frequency
	seen := make(map[rune]int)
	for _, r := range runes {
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) {
			seen[r]++
		}
	}
	for r, n := range seen {
		idx := int(hash64(string(r)) % uint64(e.dimension))
		vec[idx] += float64(n)
	}

	// Character bigram frequency
	for i := 0; i < len(runes)-1; i++ {
		if unicode.IsPunct(runes[i]) || unicode.IsSpace(runes[i]) {
			continue
		}
		idx := int(hash64(string(runes[i:i+2])) % uint64(e.dimension))
		vec[idx]++
	}

	// L2 normalize
	var mag float64
	for _, v := range vec {
		mag += v * v
	}
	mag = math.Sqrt(mag)

	out := make([]float32, e.dimension)
	if mag == 0 {
		return out, nil
	}
	for i, v := range vec {
		out[i] = float32(v / mag)
	}
	return out, nil
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// Ensure interface compliance
var _ Embedder = (*LocalEmbedder)(nil)
