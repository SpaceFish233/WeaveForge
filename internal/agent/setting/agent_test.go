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
	rlStore, err := vectordb.NewSQLiteStore(db, &mockEmbedder{vec: []float32{0.5, 0.5, 0, 0, 0}})
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

func TestDetectSettingsInChapter(t *testing.T) {
	db := inMemoryDB(t)
	db.AutoMigrate(&models.WorldSetting{})
	db.Create(&models.WorldSetting{ID: "s1", Title: "魔法体系", Content: "魔法分三类", Type: "世界观"})
	db.Create(&models.WorldSetting{ID: "s2", Title: "龙族", Content: "龙族是远古种族", Type: "种族"})
	db.Create(&models.WorldSetting{ID: "s3", Title: "", Content: "空标题设定", Type: "其他"})
	db.Create(&models.WorldSetting{ID: "s4", Title: "不存在的关键词", Content: "test", Type: "其他"})

	agent := NewAgent(nil, nil, nil, "", db)
	hits, err := agent.DetectSettingsInChapter("在这个世界中，魔法体系是核心设定，龙族是魔法体系的重要组成", true, false)
	if err != nil {
		t.Fatalf("DetectSettingsInChapter: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("want 2 hits, got %d: %+v", len(hits), hits)
	}

	found := make(map[string]bool)
	for _, h := range hits {
		found[h.Title] = true
	}
	if !found["魔法体系"] {
		t.Error("should find 魔法体系")
	}
	if !found["龙族"] {
		t.Error("should find 龙族")
	}

	// 魔法体系 出现 twice in the text
	for _, h := range hits {
		if h.Title == "魔法体系" && h.Occurrences != 2 {
			t.Errorf("魔法体系 should appear 2 times, got %d", h.Occurrences)
		}
	}

	// Empty content
	hits, err = agent.DetectSettingsInChapter("", true, false)
	if err != nil {
		t.Fatalf("empty content: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("want 0 hits for empty content, got %d", len(hits))
	}

	// Case sensitive test
	db.Create(&models.WorldSetting{ID: "s5", Title: "ABC", Content: "test", Type: "其他"})
	hits, _ = agent.DetectSettingsInChapter("abc def ABC", true, false)
	foundABC := false
	for _, h := range hits {
		if h.Title == "ABC" {
			foundABC = true
			if h.Occurrences != 1 {
				t.Errorf("case-sensitive ABC should appear 1 time, got %d", h.Occurrences)
			}
		}
	}
	if !foundABC {
		t.Error("case-sensitive should find ABC")
	}

	hitsCI, _ := agent.DetectSettingsInChapter("abc def ABC", false, false)
	foundABCCI := false
	for _, h := range hitsCI {
		if h.Title == "ABC" {
			foundABCCI = true
			if h.Occurrences != 2 {
				t.Errorf("case-insensitive ABC should appear 2 times, got %d", h.Occurrences)
			}
		}
	}
	if !foundABCCI {
		t.Error("case-insensitive should find ABC")
	}
}

func TestValidateSettingConflict_NoConflict(t *testing.T) {
	db := inMemoryDB(t)
	db.AutoMigrate(&models.WorldSetting{})
	db.Create(&models.WorldSetting{ID: "s1", Title: "魔法", Content: "主角是冰系魔法师，无法使用火系魔法", Type: "世界观"})

	agent := NewAgent(nil, nil, &mockLLM{response: `{"has_conflict": false}`}, "test", db)
	result, err := agent.ValidateSettingConflict(context.Background(),"s1", "主角使用魔法战斗")
	if err != nil {
		t.Fatalf("ValidateSettingConflict: %v", err)
	}
	if result.HasConflict {
		t.Error("should not have conflict")
	}
	if result.SettingTitle != "魔法" {
		t.Errorf("bad title: %s", result.SettingTitle)
	}
}

func TestValidateSettingConflict_WithConflict(t *testing.T) {
	db := inMemoryDB(t)
	db.AutoMigrate(&models.WorldSetting{})
	db.Create(&models.WorldSetting{ID: "s1", Title: "魔法", Content: "主角是冰系魔法师，无法使用火系魔法", Type: "世界观"})

	agent := NewAgent(nil, nil, &mockLLM{
		response: `{"has_conflict": true, "conflict_desc": "主角使用了火球术但设定为冰系", "suggested_fix": "改为冰箭术", "reference_text": "主角是冰系魔法师，无法使用火系魔法"}`,
	}, "test", db)
	result, err := agent.ValidateSettingConflict(context.Background(),"s1", "主角使用魔法发出火球术攻击敌人")
	if err != nil {
		t.Fatalf("ValidateSettingConflict: %v", err)
	}
	if !result.HasConflict {
		t.Fatal("should have conflict")
	}
	if !strings.Contains(result.ConflictDesc, "火球术") {
		t.Errorf("bad conflict_desc: %s", result.ConflictDesc)
	}
	if !strings.Contains(result.SuggestedFix, "冰箭术") {
		t.Errorf("bad suggested_fix: %s", result.SuggestedFix)
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
				runeLen := len([]rune(c))
				if runeLen > tt.maxChars+10 {
					t.Errorf("chunk %d > %d", runeLen, tt.maxChars)
				}
			}
		})
	}
}

func TestCountOccurrences_RunePositions(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		keyword   string
		wantCount int
		wantPos   []int
	}{
		{
			name:      "pure Chinese",
			text:      "在这个世界中，魔法体系是核心设定，魔法体系很重要",
			keyword:   "魔法体系",
			wantCount: 2,
			wantPos:   []int{7, 17},
		},
		{
			name:      "Chinese at start",
			text:      "魔法体系是核心",
			keyword:   "魔法体系",
			wantCount: 1,
			wantPos:   []int{0},
		},
		{
			name:      "mixed Chinese and English",
			text:      "使用Python编程语言和Go语言",
			keyword:   "Go语言",
			wantCount: 1,
			wantPos:   []int{13},
		},
		{
			name:      "English only",
			text:      "Hello World, Hello Go",
			keyword:   "Hello",
			wantCount: 2,
			wantPos:   []int{0, 13},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, positions := countOccurrences(tt.text, tt.keyword, true, false)
			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
			if len(positions) != len(tt.wantPos) {
				t.Fatalf("positions len = %d, want %d", len(positions), len(tt.wantPos))
			}
			for i, pos := range positions {
				if pos != tt.wantPos[i] {
					t.Errorf("positions[%d] = %d, want %d", i, pos, tt.wantPos[i])
				}
			}
		})
	}
}

func TestExtractSnippet_ChineseText(t *testing.T) {
	text := "在这个世界中，魔法体系是核心设定。魔法体系分为三大类：元素、精神、时空。"
	snippet := extractSnippet(text, 7, "魔法体系", 10)
	runes := []rune(snippet)
	found := false
	for i := 0; i <= len(runes)-4; i++ {
		if string(runes[i:i+4]) == "魔法体系" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("snippet should contain keyword, got: %s", snippet)
	}
	if len(runes) > 30 {
		t.Errorf("snippet too long: %d runes", len(runes))
	}
}

func TestCountOccurrences_WholeWord_RunePositions(t *testing.T) {
	text := "ABC def ABC"
	count, positions := countOccurrences(text, "ABC", true, true)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(positions) != 2 {
		t.Fatalf("positions len = %d, want 2", len(positions))
	}
	if positions[0] != 0 {
		t.Errorf("positions[0] = %d, want 0", positions[0])
	}
	if positions[1] != 8 {
		t.Errorf("positions[1] = %d, want 8", positions[1])
	}
}
