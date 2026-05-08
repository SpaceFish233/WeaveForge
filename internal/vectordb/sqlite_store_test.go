package vectordb

import (
	"context"
	"math"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockEmbedder struct {
	vec []float32
}

func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if m.vec != nil {
		return m.vec, nil
	}
	return []float32{1, 0, 0, 0}, nil
}

func inMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	return db
}

func TestSQLiteStore_AddAndSearch(t *testing.T) {
	db := inMemoryDB(t)
	store, err := NewSQLiteStore(db, &mockEmbedder{})
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	ctx := context.Background()
	docs := []Document{
		{ID: "1", Content: "魔法元素", Embedding: []float32{1, 0, 0, 0}, Metadata: `{"title":"元素","type":"世界观"}`},
		{ID: "2", Content: "剑术等级", Embedding: []float32{0, 1, 0, 0}, Metadata: `{"title":"剑术","type":"能力"}`},
		{ID: "3", Content: "龙族生物", Embedding: []float32{0, 0, 1, 0}, Metadata: `{"title":"龙族","type":"人物"}`},
		{ID: "4", Content: "帝国首都", Embedding: []float32{0, 0, 0, 1}, Metadata: `{"title":"地理","type":"地理"}`},
	}
	if err := store.AddDocuments(ctx, docs); err != nil {
		t.Fatalf("add docs: %v", err)
	}

	// SearchSimilar embeds query via mockEmbedder which returns {1,0,0,0}
	results, err := store.SearchSimilar(ctx, "魔法", 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	if results[0].ID != "1" {
		t.Errorf("want first result ID '1', got '%s'", results[0].ID)
	}
	if math.Abs(results[0].Score-1.0) > 0.001 {
		t.Errorf("want score ~1.0, got %f", results[0].Score)
	}
}

func TestSQLiteStore_Search_Empty(t *testing.T) {
	db := inMemoryDB(t)
	store, err := NewSQLiteStore(db, &mockEmbedder{})
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	ctx := context.Background()
	results, err := store.SearchSimilar(ctx, "anything", 5)
	if err != nil {
		t.Fatalf("search empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("want 0 results, got %d", len(results))
	}
}

func TestSQLiteStore_DeleteDocument(t *testing.T) {
	db := inMemoryDB(t)
	store, err := NewSQLiteStore(db, &mockEmbedder{})
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	ctx := context.Background()
	store.AddDocuments(ctx, []Document{
		{ID: "1", Content: "a", Embedding: []float32{1, 0, 0, 0}},
		{ID: "2", Content: "b", Embedding: []float32{0, 1, 0, 0}},
	})
	if err := store.DeleteDocument(ctx, "1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	results, _ := store.SearchSimilar(ctx, "b", 10)
	if len(results) != 1 || results[0].ID != "2" {
		t.Errorf("want only doc '2', got %v", results)
	}
}

func TestSQLiteStore_DeleteDocumentsByMetadata(t *testing.T) {
	db := inMemoryDB(t)
	store, _ := NewSQLiteStore(db, &mockEmbedder{})
	ctx := context.Background()
	store.AddDocuments(ctx, []Document{
		{ID: "1", Content: "a", Embedding: []float32{1, 0, 0, 0}, Metadata: `{"type":"世界观"}`},
		{ID: "2", Content: "b", Embedding: []float32{0, 1, 0, 0}, Metadata: `{"type":"能力"}`},
		{ID: "3", Content: "c", Embedding: []float32{0, 0, 1, 0}, Metadata: `{"type":"世界观"}`},
	})
	store.DeleteDocuments(ctx, map[string]string{"type": "世界观"})
	results, _ := store.SearchSimilar(ctx, "b", 10)
	if len(results) != 1 || results[0].ID != "2" {
		t.Errorf("want only doc '2', got %v", results)
	}
}

func TestCosineSimilarityF32(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"opposite", []float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0},
		{"partial", []float32{1, 1, 0}, []float32{1, 0, 0}, 0.7071067811865475},
		{"zero a", []float32{0, 0, 0}, []float32{1, 0, 0}, 0.0},
		{"different lengths", []float32{1, 0}, []float32{1, 0, 0}, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarityF32(tt.a, tt.b)
			if math.Abs(got-tt.want) > 0.0001 {
				t.Errorf("cosineSimilarityF32(%v,%v)=%f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
