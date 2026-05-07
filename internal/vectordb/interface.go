package vectordb

import "context"

// Document represents a text chunk with its vector embedding and metadata.
type Document struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	Metadata  string            `json:"metadata"` // JSON string
	Embedding []float32         `json:"-"`
	Score     float64           `json:"score,omitempty"`
}

// Embedder converts text to a vector embedding.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// VectorStore is the interface for vector storage and similarity search.
type VectorStore interface {
	AddDocuments(ctx context.Context, docs []Document) error
	// SearchSimilar embeds the query text and returns topK most similar documents.
	SearchSimilar(ctx context.Context, query string, topK int) ([]Document, error)
	// SearchSimilarFilter returns topK documents whose metadata matches all filter pairs.
	SearchSimilarFilter(ctx context.Context, query string, topK int, filter map[string]string) ([]Document, error)
	DeleteDocuments(ctx context.Context, metadata map[string]string) error
	DeleteDocument(ctx context.Context, id string) error
}
