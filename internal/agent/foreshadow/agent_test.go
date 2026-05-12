package foreshadow

import (
	"context"
	"testing"

	"weaveforge/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

func TestAutoDetectForeshadowing_RuleOnly(t *testing.T) {
	db := inMemoryDB(t)
	agent := NewAgent(db, nil, "test")

	text := "他并不知道自己的身世之谜。一个小小的玉佩竟然埋下了日后惊天变故的种子。"
	candidates, err := agent.AutoDetectForeshadowing(context.Background(), text)
	if err != nil {
		t.Fatalf("autodetect: %v", err)
	}
	if len(candidates) == 0 {
		t.Error("expected at least 1 regex candidate")
	}
	// Verify types
	for _, c := range candidates {
		if c.Text == "" {
			t.Error("candidate has empty text")
		}
	}
}

func TestConfirmForeshadowing(t *testing.T) {
	db := inMemoryDB(t)
	db.AutoMigrate(&models.Foreshadowing{})
	agent := NewAgent(db, nil, "test")
	ctx := context.Background()

	c := CandidateForeshadow{Text: "test foreshadow", Type: "人物背景", Confidence: 0.8}
	id, err := agent.ConfirmForeshadowing(ctx, c, "chapter-1")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty id")
	}

	// Verify list
	list, err := agent.ListForeshadowings(ctx, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].Status != "planted" {
		t.Errorf("expected planted, got %s", list[0].Status)
	}
}

func TestListForeshadowings_Filter(t *testing.T) {
	db := inMemoryDB(t)
	db.AutoMigrate(&models.Foreshadowing{})
	agent := NewAgent(db, nil, "test")
	ctx := context.Background()

	db.Create(&models.Foreshadowing{ID: "1", Description: "a", Status: "planted"})
	db.Create(&models.Foreshadowing{ID: "2", Description: "b", Status: "revealed"})

	list, _ := agent.ListForeshadowings(ctx, "planted")
	if len(list) != 1 {
		t.Errorf("want 1 planted, got %d", len(list))
	}
	if list[0].ID != "1" {
		t.Errorf("want ID 1, got %s", list[0].ID)
	}
}

func TestHealthReport(t *testing.T) {
	db := inMemoryDB(t)
	db.AutoMigrate(&models.Foreshadowing{})
	agent := NewAgent(db, nil, "test")
	ctx := context.Background()

	db.Create(&models.Foreshadowing{ID: "1", Description: "a", Status: "planted", Priority: "high"})
	db.Create(&models.Foreshadowing{ID: "2", Description: "b", Status: "revealed", Priority: "medium"})
	db.Create(&models.Foreshadowing{ID: "3", Description: "c", Status: "planted", Priority: "medium"})

	r, err := agent.GenerateHealthReport(ctx)
	if err != nil {
		t.Fatalf("health report: %v", err)
	}
	if r.Total != 3 {
		t.Errorf("want 3 total, got %d", r.Total)
	}
	if r.Planted != 2 {
		t.Errorf("want 2 planted, got %d", r.Planted)
	}
	if r.Revealed != 1 {
		t.Errorf("want 1 revealed, got %d", r.Revealed)
	}
	if r.PriorityCounts["high"] != 1 {
		t.Errorf("want 1 high, got %d", r.PriorityCounts["high"])
	}
}

func TestAutoDetectForeshadowing_Empty(t *testing.T) {
	agent := NewAgent(nil, nil, "test")
	candidates, err := agent.AutoDetectForeshadowing(context.Background(), "")
	if err != nil {
		t.Fatalf("autodetect empty: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("want 0, got %d", len(candidates))
	}
}

func TestForeshadowPatterns(t *testing.T) {
	tests := []struct {
		text    string
		minHits int
	}{
		{"他并不知道真相", 1},
		{"后来才发现一切都错了", 1},
		{"一个小小的戒指", 1},
		{"将在日后发挥作用", 1},
		{"埋下了隐患的种子", 1},
		{"当时没有在意这句话", 1},
		{"隐约觉得不对劲", 1},
		{"他若有所思地点点头", 1},
		{"意味深长地笑了笑", 1},
	}
	for _, tt := range tests {
		hits := 0
		for _, fp := range foreshadowPatterns {
			if fp.pattern.MatchString(tt.text) {
				hits++
			}
		}
		if hits < tt.minHits {
			t.Errorf("text %q: want >=%d hits, got %d", tt.text, tt.minHits, hits)
		}
	}
}

func TestBuildByteToRuneMap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		bytePos  int
		wantRune int
	}{
		{"ASCII", "hello", 3, 3},
		{"Chinese", "你好世界", 3, 1},   // 你 = 3 bytes
		{"Chinese byte6", "你好世界", 6, 2}, // 你好 = 6 bytes
		{"Mixed", "hi你好", 2, 2},         // hi = 2 bytes
		{"Mixed byte5", "hi你好", 5, 3},   // hi你 = 5 bytes
		{"Empty", "", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := buildByteToRuneMap(tt.input)
			if tt.bytePos < len(m) {
				if m[tt.bytePos] != tt.wantRune {
					t.Errorf("buildByteToRuneMap(%q)[%d] = %d, want %d", tt.input, tt.bytePos, m[tt.bytePos], tt.wantRune)
				}
			}
		})
	}
}

func TestAutoDetectForeshadowing_ChinesePositions(t *testing.T) {
	db := inMemoryDB(t)
	agent := NewAgent(db, nil, "test")

	// Text with Chinese characters where byte positions != rune positions
	text := "他并不知道自己的身世之谜，一个小小的玉佩埋下了祸根。"
	candidates, err := agent.AutoDetectForeshadowing(context.Background(), text)
	if err != nil {
		t.Fatalf("autodetect: %v", err)
	}

	runes := []rune(text)
	for _, c := range candidates {
		// Verify that StartIndex and EndIndex are valid rune positions
		if c.StartIndex < 0 || c.EndIndex > len(runes) || c.StartIndex >= c.EndIndex {
			t.Errorf("invalid rune positions: StartIndex=%d, EndIndex=%d, text len=%d", c.StartIndex, c.EndIndex, len(runes))
		}
		// Verify that the snippet at the positions makes sense
		if c.StartIndex < len(runes) && c.EndIndex <= len(runes) {
			snippet := string(runes[c.StartIndex:c.EndIndex])
			if snippet == "" {
				t.Errorf("empty snippet at positions [%d:%d]", c.StartIndex, c.EndIndex)
			}
		}
	}
}
