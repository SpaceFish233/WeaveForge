package setting

import (
	"context"
	"strings"
	"testing"

	"weaveforge/internal/llm"
	"weaveforge/internal/vectordb"
	"weaveforge/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) ChatCompletion(_ context.Context, messages []llm.Message, model string, opts ...llm.ChatOption) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

type mockStore struct {
	vectordb.VectorStore
	docs []vectordb.Document
}

func (m *mockStore) SearchSimilar(_ context.Context, query string, topK int) ([]vectordb.Document, error) {
	if len(m.docs) == 0 {
		return nil, nil
	}
	n := topK
	if n > len(m.docs) {
		n = len(m.docs)
	}
	r := make([]vectordb.Document, n)
	copy(r, m.docs[:n])
	for i := range r {
		r[i].Score = 0.95 - float64(i)*0.1
	}
	return r, nil
}
func (m *mockStore) AddDocuments(_ context.Context, docs []vectordb.Document) error {
	m.docs = append(m.docs, docs...)
	return nil
}
func (m *mockStore) DeleteDocuments(_ context.Context, metadata map[string]string) error { return nil }
func (m *mockStore) DeleteDocument(_ context.Context, id string) error                  { return nil }
func (m *mockStore) SearchSimilarFilter(_ context.Context, query string, topK int, filter map[string]string) ([]vectordb.Document, error) {
	return m.SearchSimilar(nil, query, topK)
}

type mockEmbedder struct {
	vec []float32
}

func (m *mockEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if m.vec != nil {
		return m.vec, nil
	}
	return []float32{1, 0, 0, 0, 0}, nil
}

func inMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func TestUploadWorldSetting(t *testing.T) {
	db := inMemoryDB(t)
	db.AutoMigrate(&models.WorldSetting{})
	rlStore, err := vectordb.NewSQLiteStore(db, &mockEmbedder{vec: []float32{0.5, 0.5, 0, 0, 0}}, 5)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	agent := NewAgent(rlStore, &mockEmbedder{}, &mockLLM{}, "test-model", db)
	ctx := context.Background()
	if err := agent.UploadWorldSetting(ctx, "魔法体系", "魔法分为三大类：元素、精神、时空。", "世界观"); err != nil {
		t.Fatalf("UploadWorldSetting: %v", err)
	}
	var count int64
	db.Model(&models.WorldSetting{}).Count(&count)
	if count != 1 {
		t.Errorf("want 1 setting, got %d", count)
	}
}

func TestCheckConsistency_NoConflict(t *testing.T) {
	agent := NewAgent(
		&mockStore{docs: []vectordb.Document{
			{Content: "主角拥有火系魔法天赋", Metadata: `{"title":"主角设定"}`},
		}}, &mockEmbedder{}, &mockLLM{response: `[]`}, "test", nil)
	w, err := agent.CheckConsistency(context.Background(), "主角使用火球术")
	if err != nil {
		t.Fatalf("CheckConsistency: %v", err)
	}
	if len(w) != 0 {
		t.Errorf("want 0 conflicts, got %d: %v", len(w), w)
	}
}

func TestCheckConsistency_WithConflict(t *testing.T) {
	agent := NewAgent(
		&mockStore{docs: []vectordb.Document{
			{Content: "主角是冰系魔法师，无法使用火系魔法", Metadata: `{"title":"角色设定"}`},
		}}, &mockEmbedder{}, &mockLLM{response: `[{"conflict_desc":"主角用了火球术但设定是冰系","suggested_fix":"改为冰箭术","reference_text":"主角是冰系魔法师"}]`}, "test", nil)
	w, err := agent.CheckConsistency(context.Background(), "主角使用火球术攻击")
	if err != nil {
		t.Fatalf("CheckConsistency: %v", err)
	}
	if len(w) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(w))
	}
	if !strings.Contains(w[0].ConflictDesc, "火球术") {
		t.Errorf("bad conflict_desc: %s", w[0].ConflictDesc)
	}
	if !strings.Contains(w[0].SuggestedFix, "冰箭术") {
		t.Errorf("bad suggested_fix: %s", w[0].SuggestedFix)
	}
}

func TestCheckConsistency_EmptyContent(t *testing.T) {
	agent := NewAgent(&mockStore{}, &mockEmbedder{}, &mockLLM{}, "test", nil)
	w, _ := agent.CheckConsistency(context.Background(), "")
	if len(w) != 0 {
		t.Errorf("want 0, got %d", len(w))
	}
	w, _ = agent.CheckConsistency(context.Background(), "   ")
	if len(w) != 0 {
		t.Errorf("want 0, got %d", len(w))
	}
}

func TestChunkText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxChars int
		min      int
	}{
		{"empty", "", 100, 0},
		{"short", "Hello World", 100, 1},
		{"long", strings.Repeat("这是一个测试句子。", 50), 50, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkText(tt.input, tt.maxChars)
			if len(chunks) < tt.min {
				t.Errorf("want >= %d chunks, got %d", tt.min, len(chunks))
			}
			for _, c := range chunks {
				if len([]rune(c)) > tt.maxChars+10 {
					t.Errorf("chunk %d > %d", len([]rune(c)), tt.maxChars)
				}
			}
		})
	}
}

func TestParseLLMResponse(t *testing.T) {
	tests := []struct {
		name    string
		resp    string
		wantLen int
		wantErr bool
	}{
		{"empty array", `[]`, 0, false},
		{"one", `[{"conflict_desc":"d","suggested_fix":"f","reference_text":"r"}]`, 1, false},
		{"multiple", `[{"conflict_desc":"d1","suggested_fix":"f1","reference_text":"r1"},{"conflict_desc":"d2","suggested_fix":"f2","reference_text":"r2"}]`, 2, false},
		{"with surrounding", `检查发现：[{"conflict_desc":"d","suggested_fix":"f","reference_text":"r"}] 请修改`, 1, false},
		{"no JSON", `没有冲突`, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := parseLLMResponse(tt.resp, nil)
			if tt.wantErr {
				if err == nil {
					t.Error("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(r) != tt.wantLen {
				t.Errorf("want %d, got %d", tt.wantLen, len(r))
			}
		})
	}
}
