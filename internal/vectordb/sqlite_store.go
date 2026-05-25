package vectordb

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type vectorDoc struct {
	ID        string    `gorm:"primaryKey;size:36"`
	Content   string    `gorm:"type:text"`
	Metadata  string    `gorm:"type:text"`
	Embedding []byte    `gorm:"type:blob"`
	CreatedAt time.Time
}

// SQLiteStore implements VectorStore using SQLite with in-memory similarity search.
// It requires an Embedder to convert query text to vectors at search time.
type SQLiteStore struct {
	db    *gorm.DB
	embed Embedder
	mu    sync.RWMutex
}

func NewSQLiteStore(db *gorm.DB, embed Embedder) (*SQLiteStore, error) {
	if err := db.AutoMigrate(&vectorDoc{}); err != nil {
		return nil, fmt.Errorf("vectordb: migrate: %w", err)
	}
	return &SQLiteStore{db: db, embed: embed}, nil
}

func floats32ToBytes(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func bytesToFloats32(data []byte) []float32 {
	if len(data) == 0 {
		return nil
	}
	vec := make([]float32, len(data)/4)
	for i := range vec {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return vec
}

func cosineSimilarityF32(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (s *SQLiteStore) AddDocuments(ctx context.Context, docs []Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range docs {
		if d.Metadata == "" {
			d.Metadata = "{}"
		}
		// Validate it's valid JSON
		if !json.Valid([]byte(d.Metadata)) {
			return fmt.Errorf("vectordb: invalid JSON in metadata")
		}

		id := d.ID
		if id == "" {
			id = uuid.New().String()
		}

		doc := vectorDoc{
			ID:        id,
			Content:   d.Content,
			Metadata:  d.Metadata,
			Embedding: floats32ToBytes(d.Embedding),
			CreatedAt: time.Now(),
		}
		if err := s.db.WithContext(ctx).Create(&doc).Error; err != nil {
			return fmt.Errorf("vectordb: insert: %w", err)
		}
	}
	return nil
}

// maxSearchDocs limits the number of documents loaded into memory for similarity search.
const maxSearchDocs = 10000

func (s *SQLiteStore) SearchSimilar(ctx context.Context, query string, topK int) ([]Document, error) {
	// Step 1: embed the query text (outside lock to avoid blocking writes)
	queryVec, err := s.embed.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("vectordb: embed query: %w", err)
	}

	s.mu.RLock()
	var rows []vectorDoc
	if err := s.db.WithContext(ctx).Limit(maxSearchDocs).Find(&rows).Error; err != nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("vectordb: query: %w", err)
	}
	s.mu.RUnlock()

	type scored struct {
		doc vectorDoc
		sim float64
	}
	var results []scored
	for _, row := range rows {
		vec := bytesToFloats32(row.Embedding)
		sim := cosineSimilarityF32(queryVec, vec)
		results = append(results, scored{row, sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].sim > results[j].sim
	})
	if topK > 0 && topK < len(results) {
		results = results[:topK]
	}

	docs := make([]Document, len(results))
	for i, r := range results {
		metaStr := r.doc.Metadata
		if metaStr == "" {
			metaStr = "{}"
		}
		docs[i] = Document{
			ID:        r.doc.ID,
			Content:   r.doc.Content,
			Metadata:  metaStr,
			Embedding: bytesToFloats32(r.doc.Embedding),
			Score:     r.sim,
		}
	}
	return docs, nil
}

func (s *SQLiteStore) SearchSimilarFilter(ctx context.Context, query string, topK int, filter map[string]string) ([]Document, error) {
	all, err := s.SearchSimilar(ctx, query, 100)
	if err != nil {
		return nil, err
	}
	var filtered []Document
	for _, d := range all {
		var meta map[string]string
		if err := json.Unmarshal([]byte(d.Metadata), &meta); err != nil {
			continue
		}
		match := true
		for k, v := range filter {
			if meta[k] != v {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, d)
		}
	}
	if topK > 0 && topK < len(filtered) {
		filtered = filtered[:topK]
	}
	return filtered, nil
}

func (s *SQLiteStore) DeleteDocuments(ctx context.Context, metadata map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var rows []vectorDoc
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		var meta map[string]string
		if err := json.Unmarshal([]byte(row.Metadata), &meta); err != nil {
			continue
		}
		match := true
		for k, v := range metadata {
			if meta[k] != v {
				match = false
				break
			}
		}
		if match {
			if err := s.db.WithContext(ctx).Delete(&vectorDoc{}, "id = ?", row.ID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SQLiteStore) DeleteDocument(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.WithContext(ctx).Delete(&vectorDoc{}, "id = ?", id).Error
}
