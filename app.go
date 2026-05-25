package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"weaveforge/internal/agent/character"
	"weaveforge/internal/agent/foreshadow"
	"weaveforge/internal/agent/plotengine"
	"weaveforge/internal/agent/outline"
	"weaveforge/internal/agent/relationship"
	"weaveforge/internal/agent/setting"
	"weaveforge/internal/agent/stats"
	"weaveforge/internal/agent/style"
	"weaveforge/internal/agent/timeline"
	"weaveforge/internal/agent/typo"
	"weaveforge/internal/config"
	"weaveforge/internal/coordinator"
	"weaveforge/internal/llm"
	"weaveforge/internal/vectordb"
	"weaveforge/models"
	"weaveforge/parser"
	"weaveforge/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx              context.Context
	chapter          *services.ChapterService
	volume           *services.VolumeService
	search           *services.SearchService
	characterAgent   *character.Agent
	settingAgent     *setting.Agent
	styleAgent       *style.Agent
	foreshadowAgent  *foreshadow.Agent
	plotEngine       *plotengine.Agent
	relationshipAgent *relationship.Agent
	outlineAgent     *outline.Agent
	timelineAgent    *timeline.Agent
	typoAgent        *typo.Agent
	statsAgent       *stats.Agent
	coordinator      *coordinator.Coordinator
	appConfig        *config.Config
	configMu         sync.RWMutex
	embedder         vectordb.Embedder
}

func NewApp(cs *services.ChapterService, vs *services.VolumeService, ss *services.SearchService, ca *character.Agent, sa *setting.Agent, sta *style.Agent, fa *foreshadow.Agent, pe *plotengine.Agent, rla *relationship.Agent, ola *outline.Agent, tla *timeline.Agent, ta *typo.Agent, stAgent *stats.Agent, co *coordinator.Coordinator, cfg *config.Config, emb vectordb.Embedder) *App {
	return &App{chapter: cs, volume: vs, search: ss, characterAgent: ca, settingAgent: sa, styleAgent: sta, foreshadowAgent: fa, plotEngine: pe, relationshipAgent: rla, outlineAgent: ola, timelineAgent: tla, typoAgent: ta, statsAgent: stAgent, coordinator: co, appConfig: cfg, embedder: emb}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.coordinator != nil {
		a.coordinator.SetContext(ctx)
	}
}

func (a *App) shutdown(_ context.Context) {
	if a.coordinator != nil {
		a.coordinator.Stop()
	}
}

// ─── Chapter API ────────────────────────────────────────────────────

func (a *App) CreateChapter(title, content, volumeID string) (string, error) { return a.chapter.CreateChapter(title, content, volumeID) }
func (a *App) UpdateChapter(chapterID, content string) error {
	// Get old content for stats tracking
	oldChapter, err := a.chapter.GetChapter(chapterID)
	oldContent := ""
	if err == nil {
		oldContent = oldChapter.Content
	}

	if err := a.chapter.UpdateChapter(chapterID, content); err != nil {
		return err
	}

	// Update daily stats in background (non-blocking)
	// Use a fresh background context with timeout to avoid depending on a.ctx lifecycle.
	if a.statsAgent != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := a.statsAgent.UpdateDailyStats(ctx, chapterID, oldContent, content); err != nil {
				log.Printf("UpdateDailyStats failed: %v", err)
			}
		}()
	}

	return nil
}
func (a *App) UpdateChapterTitle(chapterID, title string) error              { return a.chapter.UpdateChapterTitle(chapterID, title) }
func (a *App) UpdateChapterVolume(chapterID, volumeID string) error          { return a.chapter.UpdateChapterVolume(chapterID, volumeID) }
func (a *App) DeleteChapter(chapterID string) error                          { return a.chapter.DeleteChapter(chapterID) }
func (a *App) GetChapter(chapterID string) (models.Chapter, error)           { return a.chapter.GetChapter(chapterID) }
func (a *App) ListChapters() ([]models.ChapterSummary, error)                { return a.chapter.ListChapters() }
func (a *App) ReorderChapters(chapterIDs []string) error                     { return a.chapter.ReorderChapters(chapterIDs) }

func (a *App) ImportDocument(filePath string) ([]string, error) {
	cleanPath := filepath.Clean(filePath)
	chs, err := parser.ParseFile(cleanPath)
	if err != nil {
		return nil, err
	}
	return a.importChapters(chs)
}
func (a *App) ImportContent(filename, content string) ([]string, error) {
	const maxContentBytes = 10 * 1024 * 1024 // 10 MB
	if len(content) > maxContentBytes {
		return nil, fmt.Errorf("content too large: %d bytes (max %d)", len(content), maxContentBytes)
	}
	ext := ""
	if idx := strings.LastIndex(filename, "."); idx >= 0 {
		ext = filename[idx:]
	}
	return a.importChapters(parser.ParseContent(content, ext))
}
func (a *App) importChapters(chs []parser.ParsedChapter) ([]string, error) { var ids []string; for _, ch := range chs { id, err := a.chapter.CreateChapter(ch.Title, ch.Content, ""); if err != nil { return ids, err }; ids = append(ids, id) }; return ids, nil }

// ─── Volume API ─────────────────────────────────────────────────────

func (a *App) CreateVolume(name string) (string, error)       { return a.volume.CreateVolume(name) }
func (a *App) ListVolumes() ([]models.VolumeSummary, error)    { return a.volume.ListVolumes() }
func (a *App) UpdateVolume(id, name string) error              { return a.volume.UpdateVolume(id, name) }
func (a *App) DeleteVolume(id string) error                    { return a.volume.DeleteVolume(id) }

// ─── Character API ─────────────────────────────────────────────────

func (a *App) CreateCharacter(c models.Character) (string, error) { return a.characterAgent.CreateCharacter(a.ctx, c) }
func (a *App) ListCharacters() ([]character.CharacterSummary, error) { return a.characterAgent.ListCharacters(a.ctx) }
func (a *App) GetCharacter(id string) (*models.Character, error) { return a.characterAgent.GetCharacter(a.ctx, id) }
func (a *App) UpdateCharacter(id string, c models.Character) error { return a.characterAgent.UpdateCharacter(a.ctx, id, c) }
func (a *App) DeleteCharacter(id string) error { return a.characterAgent.DeleteCharacter(a.ctx, id) }

// ─── World Setting API ──────────────────────────────────────────────

func (a *App) UploadWorldSetting(title, content, settingType string) error { return a.settingAgent.UploadWorldSetting(a.ctx, title, content, settingType) }
func (a *App) ListSettings() ([]setting.SettingInfo, error)                { return a.settingAgent.ListSettings(a.ctx) }
func (a *App) GetSetting(id string) (*models.WorldSetting, error)         { return a.settingAgent.GetSetting(a.ctx, id) }
func (a *App) UpdateSetting(id, title, content, settingType string) error  { return a.settingAgent.UpdateSetting(a.ctx, id, title, content, settingType) }
func (a *App) DeleteSetting(id string) error                               { return a.settingAgent.DeleteSetting(a.ctx, id) }
func (a *App) DetectSettings(chapterContent string, caseSensitive, wholeWord bool) ([]setting.SettingHit, error) { return a.settingAgent.DetectSettingsInChapter(chapterContent, caseSensitive, wholeWord) }
func (a *App) ValidateSetting(settingID, chapterContent string) (*setting.ConflictResult, error) { return a.settingAgent.ValidateSettingConflict(a.ctx, settingID, chapterContent) }

// ─── Polish API ─────────────────────────────────────────────────────

func (a *App) PolishWithInstruction(text, instruction, profileID string) (string, error) {
	return a.styleAgent.PolishWithInstruction(a.ctx, text, instruction, profileID)
}

// ─── Anti-AI Flavor Detection API ─────────────────────────────────────

func (a *App) DetectAIFlavor(text string) (style.AIFlavorReport, error) {
	return a.styleAgent.DetectAIFlavor(a.ctx, text)
}

// ─── Chapter Hook Check API ───────────────────────────────────────────

func (a *App) CheckChapterHook(chapterContent, prevChapterContent string) (style.HookCheckResult, error) {
	return a.styleAgent.CheckChapterHook(a.ctx, chapterContent, prevChapterContent)
}

// ─── Pre-Write Constraint Check API ───────────────────────────────────

func (a *App) CheckWritingConstraints(chapterContent string) (style.ConstraintCheckResult, error) {
	return a.styleAgent.CheckWritingConstraints(a.ctx, chapterContent)
}

// ─── Placeholder Scan API ─────────────────────────────────────────────

func (a *App) ScanPlaceholders(text string) style.PlaceholderScanResult {
	return a.styleAgent.ScanPlaceholders(text)
}

// ─── Foreshadow Agent API ───────────────────────────────────────────

func (a *App) AutoDetectForeshadowing(chapterContent string) ([]foreshadow.CandidateForeshadow, error) { return a.foreshadowAgent.AutoDetectForeshadowing(a.ctx, chapterContent) }
func (a *App) ConfirmForeshadowing(candidate foreshadow.CandidateForeshadow, chapterID string) (string, error) { return a.foreshadowAgent.ConfirmForeshadowing(a.ctx, candidate, chapterID) }
func (a *App) SaveForeshadowing(text string, chapterID string) (string, error) { return a.foreshadowAgent.SaveForeshadowing(a.ctx, text, chapterID) }
func (a *App) ListForeshadowings(statusFilter string) ([]foreshadow.ForeshadowSummary, error) { return a.foreshadowAgent.ListForeshadowings(a.ctx, statusFilter) }
func (a *App) GetForeshadowing(id string) (*models.Foreshadowing, error) { return a.foreshadowAgent.GetForeshadowing(a.ctx, id) }
func (a *App) UpdateForeshadowing(id string, updates map[string]interface{}) error { return a.foreshadowAgent.UpdateForeshadowing(a.ctx, id, updates) }
func (a *App) DeleteForeshadowing(id string) error { return a.foreshadowAgent.DeleteForeshadowing(a.ctx, id) }
func (a *App) SuggestReveal(currentChapterIndex int) ([]foreshadow.RevealSuggestion, error) { return a.foreshadowAgent.SuggestReveal(a.ctx, currentChapterIndex) }
func (a *App) GenerateHealthReport() (*foreshadow.HealthReport, error) { return a.foreshadowAgent.GenerateHealthReport(a.ctx) }

// ─── Plot Engine API ────────────────────────────────────────────────

func (a *App) GenerateBranches(input plotengine.GenerationParams) ([]plotengine.Branch, error) { return a.plotEngine.GenerateBranches(a.ctx, input) }
func (a *App) AnalyseBranch(branch plotengine.Branch) (*plotengine.BranchAnalysis, error) { return a.plotEngine.AnalyseBranch(a.ctx, branch) }
func (a *App) MergeBranches(selectedPoints []string) (string, error) { return a.plotEngine.MergeBranches(a.ctx, selectedPoints) }
func (a *App) GenerateDialogue(charactersJSON, plotSummary string) (string, error) { return a.plotEngine.GenerateDialogue(a.ctx, charactersJSON, plotSummary) }
func (a *App) ReviseDialogue(originalDialogue, revisionPrompt string) (string, error) { return a.plotEngine.ReviseDialogue(a.ctx, originalDialogue, revisionPrompt) }

// ─── Relationship API ───────────────────────────────────────────────

func (a *App) GetRelationships(filterChapterID string) ([]relationship.RelationshipSummary, error) { return a.relationshipAgent.GetRelationships(a.ctx, filterChapterID) }
func (a *App) CreateRelationship(charAID, charBID, relType, startChapterID, note string) (string, error) { return a.relationshipAgent.CreateRelationship(a.ctx, charAID, charBID, relType, startChapterID, note) }
func (a *App) UpdateRelationship(id, relType, startChapterID, endChapterID, note string) error { return a.relationshipAgent.UpdateRelationship(a.ctx, id, relType, startChapterID, endChapterID, note) }
func (a *App) DeleteRelationship(id string) error { return a.relationshipAgent.DeleteRelationship(a.ctx, id) }
func (a *App) GetRelationshipGraph(filterChapterID string) (*relationship.GraphData, error) { return a.relationshipAgent.GetGraphData(a.ctx, filterChapterID) }
func (a *App) SaveGraphPositions(positions []relationship.NodePosition) error { return a.relationshipAgent.SaveNodePositions(a.ctx, positions) }

// ─── Search API ─────────────────────────────────────────────────────

func (a *App) SearchChapters(keyword, scope string, caseSensitive, wholeWord, useRegex bool, currentChapterID, currentVolumeID string) ([]services.SearchResult, error) {
	return a.search.SearchAll(keyword, scope, caseSensitive, wholeWord, useRegex, currentChapterID, currentVolumeID)
}

func (a *App) ReplaceInChapters(replacements []services.ReplaceItem, replacement string) (int, error) {
	return a.search.BatchReplace(replacements, replacement)
}

// ─── Outline API ────────────────────────────────────────────────────

func (a *App) GetOutlineTree() ([]outline.OutlineTreeNode, error) { return a.outlineAgent.GetTree(a.ctx) }
func (a *App) CreateOutlineNode(parentID, title, summary string) (string, error) { return a.outlineAgent.CreateNode(a.ctx, parentID, title, summary) }
func (a *App) UpdateOutlineNode(id, title, summary, status string) error { return a.outlineAgent.UpdateNode(a.ctx, id, title, summary, status) }
func (a *App) DeleteOutlineNode(id string) error { return a.outlineAgent.DeleteNode(a.ctx, id) }
func (a *App) MoveOutlineNode(id, newParentID string, newSortOrder int) error { return a.outlineAgent.MoveNode(a.ctx, id, newParentID, newSortOrder) }
func (a *App) BindOutlineChapter(nodeID, chapterID string) error { return a.outlineAgent.BindChapter(a.ctx, nodeID, chapterID) }
func (a *App) UnbindOutlineChapter(nodeID string) error { return a.outlineAgent.UnbindChapter(a.ctx, nodeID) }
func (a *App) ImportOutlineFromChapters() error { return a.outlineAgent.ImportFromChapters(a.ctx) }
func (a *App) ExportOutlineMarkdown() (string, error) { return a.outlineAgent.ExportMarkdown(a.ctx) }
func (a *App) GetOutlineNodeByChapter(chapterID string) (*outline.OutlineTreeNode, error) { return a.outlineAgent.GetNodeByChapterID(a.ctx, chapterID) }

// ─── Timeline API ───────────────────────────────────────────────────

func (a *App) GetTimeBase() (*models.TimeBase, error) { return a.timelineAgent.GetTimeBase(a.ctx) }
func (a *App) SaveTimeBase(description, unit string) (*models.TimeBase, error) { return a.timelineAgent.SaveTimeBase(a.ctx, description, unit) }
func (a *App) ListTimelineNodes() ([]timeline.NodeWithEvents, error) { return a.timelineAgent.ListNodes(a.ctx) }
func (a *App) GetTimelineNode(id string) (*timeline.NodeWithEvents, error) { return a.timelineAgent.GetNode(a.ctx, id) }
func (a *App) CreateTimelineNode(offsetDays float64, label, description string) (string, error) { return a.timelineAgent.CreateNode(a.ctx, offsetDays, label, description) }
func (a *App) UpdateTimelineNode(id string, offsetDays float64, label, description string) error { return a.timelineAgent.UpdateNode(a.ctx, id, offsetDays, label, description) }
func (a *App) DeleteTimelineNode(id string) error { return a.timelineAgent.DeleteNode(a.ctx, id) }
func (a *App) CreateTimelineEvent(nodeID, title, summary string, chapterIDs []string, isGradual bool, eventName, rawTimeExpr string) (string, error) { return a.timelineAgent.CreateEvent(a.ctx, nodeID, title, summary, chapterIDs, isGradual, eventName, rawTimeExpr) }
func (a *App) UpdateTimelineEvent(id, title, summary string, chapterIDs []string, isGradual bool, eventName string) error { return a.timelineAgent.UpdateEvent(a.ctx, id, title, summary, chapterIDs, isGradual, eventName) }
func (a *App) DeleteTimelineEvent(id string) error { return a.timelineAgent.DeleteEvent(a.ctx, id) }
func (a *App) ScanChaptersForTimeline(chapterIDs []string) (*timeline.ScanResult, error) { return a.timelineAgent.ScanChapters(a.ctx, chapterIDs) }

// ─── Typo Detection API ───────────────────────────────────────────────

func (a *App) DetectTypos(chapterContent string) ([]typo.TypoSuggestion, error) { return a.typoAgent.DetectTypos(a.ctx, chapterContent) }

// ─── Writing Stats API ───────────────────────────────────────────────

func (a *App) GetWritingStats(days int) ([]stats.DailyStatsRow, error) {
	return a.statsAgent.GetStats(a.ctx, days)
}

func (a *App) GetWritingStats365() ([]stats.DailyStatsRow, error) {
	return a.statsAgent.GetStats365(a.ctx)
}

func (a *App) GetTodayWritingStats() (*stats.DailyStatsRow, error) {
	return a.statsAgent.GetTodayStats(a.ctx)
}

func (a *App) GetWeekWritingStats() (int, error) {
	return a.statsAgent.GetWeekStats(a.ctx)
}

func (a *App) GetWritingStreak() (int, error) {
	return a.statsAgent.GetStreak(a.ctx)
}

func (a *App) SetWritingGoals(dailyGoal, weeklyGoal int) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	a.appConfig.Stats.DailyGoal = dailyGoal
	a.appConfig.Stats.WeeklyGoal = weeklyGoal
	return config.Save(a.appConfig)
}

func (a *App) GetWritingGoals() *stats.GoalSettings {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return &stats.GoalSettings{
		DailyGoal:      a.appConfig.Stats.DailyGoal,
		WeeklyGoal:     a.appConfig.Stats.WeeklyGoal,
		StreakWarnDays: a.appConfig.Stats.StreakWarnDays,
	}
}

func (a *App) UpdateWritingGoalSettings(streakWarnDays int) error {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	a.appConfig.Stats.StreakWarnDays = streakWarnDays
	return config.Save(a.appConfig)
}

func (a *App) ExportStatsCSV() (string, error) {
	return a.statsAgent.ExportCSV(a.ctx)
}

// ─── Config API ─────────────────────────────────────────────────────

func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	runes := []rune(key)
	if len(runes) <= 4 {
		return string(runes) + "****"
	}
	return string(runes[:4]) + "****"
}

func (a *App) GetConfig() *config.Config {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	// Return a copy (API key is already decrypted in memory, type=password hides it in the UI)
	copy := *a.appConfig
	return &copy
}
func (a *App) UpdateConfig(cfg *config.Config) error {
	// If the submitted API key is the masked value, preserve the original key
	a.configMu.RLock()
	originalKey := a.appConfig.LLM.APIKey
	a.configMu.RUnlock()
	masked := maskAPIKey(originalKey)
	if cfg.LLM.APIKey == masked || cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = originalKey
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	a.configMu.Lock()
	a.appConfig = cfg
	a.configMu.Unlock()
	return nil
}

func (a *App) TestLLMConnection(baseURL string, apiKey string) error {
	client := llm.NewClient(baseURL, apiKey)
	_, err := client.ChatCompletion(a.ctx, []llm.Message{{Role: "user", Content: "Say OK"}}, "gpt-4o-mini", llm.ChatOption{MaxTokens: 10, Temperature: 0})
	if err != nil { _, err2 := client.ChatCompletion(a.ctx, []llm.Message{{Role: "user", Content: "Say OK"}}, "deepseek-chat", llm.ChatOption{MaxTokens: 10, Temperature: 0}); if err2 != nil { return err } }; return nil
}
func (a *App) TestEmbeddingConnection(baseURL string, apiKey string) error { client := llm.NewClient(baseURL, apiKey); _, err := client.GetEmbedding(a.ctx, "test", "text-embedding-3-small"); return err }

func (a *App) TestEmbeddingLocal() (string, error) {
	// Semantic similarity test: two similar sentences vs one unrelated sentence.
	// A working embedding model should produce much higher cosine similarity for the similar pair.
	vecA, err := a.embedder.Embed(a.ctx, "今天天气真好，阳光明媚")
	if err != nil {
		return "", err
	}
	vecB, err := a.embedder.Embed(a.ctx, "晴朗的天空万里无云")
	if err != nil {
		return "", err
	}
	vecC, err := a.embedder.Embed(a.ctx, "计算机编程语言Python")
	if err != nil {
		return "", err
	}

	engine := a.getEmbedderEngine()
	simAB := cosineSimilarity(vecA, vecB)
	simAC := cosineSimilarity(vecA, vecC)
	gap := simAB - simAC

	var verdict string
	if gap > 0.3 {
		verdict = "✓ 区分度良好，模型工作正常"
	} else if gap > 0.1 {
		verdict = "△ 区分度较弱（hash 引擎正常表现，或模型质量有限）"
	} else if gap > 0 {
		verdict = "△ 区分度极弱，语义嵌入效果有限"
	} else {
		verdict = "✗ 异常：相近句相似度不高于无关句，模型可能未正确加载"
	}

	preview := formatVectorPreview(vecA)
	return fmt.Sprintf("引擎: %s, 输出维度: %d\n相似度: 相近句=%.4f, 无关句=%.4f, 区分度=%.4f\n%s\n前几个值: %s",
		engine, len(vecA), simAB, simAC, gap, verdict, preview), nil
}

func cosineSimilarity(a, b []float32) float64 {
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

func formatVectorPreview(vec []float32) string {
	n := len(vec)
	if n > 3 {
		n = 3
	}
	if n == 0 {
		return "[]"
	}
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		parts[i] = fmt.Sprintf("%.4f", vec[i])
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (a *App) getEmbedderEngine() string {
	if fe, ok := a.embedder.(*vectordb.FallbackEmbedder); ok {
		return fe.Engine()
	}
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return a.appConfig.Embedding.Engine
}

func (a *App) GetEmbedderEngine() string {
	return a.getEmbedderEngine()
}

// ─── Coordinator API ───────────────────────────────────────────────

func (a *App) OnParagraphWritten(chapterID, paragraphText string) { a.coordinator.OnParagraphWritten(chapterID, paragraphText) }
func (a *App) SetAssistantIntensity(level int) { a.coordinator.SetIntensity(level) }
func (a *App) GetAssistantIntensity() int { return a.coordinator.GetIntensity() }
func (a *App) GetSessionHistory() []coordinator.SessionEvent { return a.coordinator.GetSessionHistory() }
func (a *App) RecordNotificationAction(notifID, action string) { a.coordinator.RecordUserAction(notifID, action) }

// ─── File Dialog API ────────────────────────────────────────────────

func (a *App) SelectExeFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 llama-server 可执行文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "可执行文件 (*.exe)", Pattern: "*.exe"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	return path, err
}

func (a *App) SelectGGUFFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 GGUF 模型文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "GGUF 模型 (*.gguf)", Pattern: "*.gguf"},
			{DisplayName: "所有文件 (*.*)", Pattern: "*.*"},
		},
	})
	return path, err
}
