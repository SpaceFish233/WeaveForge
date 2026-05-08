package main

import (
	"context"
	"fmt"
	"math"
	"strings"

	"weaveforge/internal/agent/character"
	"weaveforge/internal/agent/foreshadow"
	"weaveforge/internal/agent/plotengine"
	"weaveforge/internal/agent/setting"
	"weaveforge/internal/agent/style"
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
	characterAgent   *character.Agent
	settingAgent     *setting.Agent
	styleAgent       *style.Agent
	foreshadowAgent  *foreshadow.Agent
	plotEngine       *plotengine.Agent
	coordinator      *coordinator.Coordinator
	appConfig        *config.Config
	embedder         vectordb.Embedder
}

func NewApp(cs *services.ChapterService, vs *services.VolumeService, ca *character.Agent, sa *setting.Agent, sta *style.Agent, fa *foreshadow.Agent, pe *plotengine.Agent, co *coordinator.Coordinator, cfg *config.Config, emb vectordb.Embedder) *App {
	return &App{chapter: cs, volume: vs, characterAgent: ca, settingAgent: sa, styleAgent: sta, foreshadowAgent: fa, plotEngine: pe, coordinator: co, appConfig: cfg, embedder: emb}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// ─── Chapter API ────────────────────────────────────────────────────

func (a *App) CreateChapter(title, content, volumeID string) (string, error) { return a.chapter.CreateChapter(title, content, volumeID) }
func (a *App) UpdateChapter(chapterID, content string) error                 { return a.chapter.UpdateChapter(chapterID, content) }
func (a *App) UpdateChapterTitle(chapterID, title string) error              { return a.chapter.UpdateChapterTitle(chapterID, title) }
func (a *App) UpdateChapterVolume(chapterID, volumeID string) error          { return a.chapter.UpdateChapterVolume(chapterID, volumeID) }
func (a *App) DeleteChapter(chapterID string) error                          { return a.chapter.DeleteChapter(chapterID) }
func (a *App) GetChapter(chapterID string) (models.Chapter, error)           { return a.chapter.GetChapter(chapterID) }
func (a *App) ListChapters() ([]models.ChapterSummary, error)                { return a.chapter.ListChapters() }
func (a *App) ReorderChapters(chapterIDs []string) error                     { return a.chapter.ReorderChapters(chapterIDs) }

func (a *App) ImportDocument(filePath string) ([]string, error) { chs, err := parser.ParseFile(filePath); if err != nil { return nil, err }; return a.importChapters(chs) }
func (a *App) ImportContent(filename, content string) ([]string, error) { ext := ""; if idx := strings.LastIndex(filename, "."); idx >= 0 { ext = filename[idx:] }; return a.importChapters(parser.ParseContent(content, ext)) }
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
func (a *App) ValidateSetting(settingID, chapterContent string) (*setting.ConflictResult, error) { return a.settingAgent.ValidateSettingConflict(settingID, chapterContent) }

// ─── Style Agent API ────────────────────────────────────────────────

func (a *App) LearnStyle(name string, chapterIDs []string) (string, error)        { return a.styleAgent.LearnStyle(a.ctx, name, chapterIDs) }
func (a *App) ListStyleProfiles() ([]style.StyleProfileSummary, error)            { return a.styleAgent.ListProfiles(a.ctx) }
func (a *App) GetStyleProfile(id string) (*models.StyleProfile, error)            { return a.styleAgent.GetProfile(a.ctx, id) }
func (a *App) DeleteStyleProfile(id string) error                                  { return a.styleAgent.DeleteProfile(a.ctx, id) }
func (a *App) PolishText(text, profileID, intensity string) (string, error)       { return a.styleAgent.PolishText(a.ctx, text, profileID, intensity) }
func (a *App) AnalyzeStyle(text, profileID string) (string, error)                { return a.styleAgent.AnalyzeStyle(a.ctx, text, profileID) }

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

// ─── Config API ─────────────────────────────────────────────────────

func (a *App) GetConfig() *config.Config { return a.appConfig }
func (a *App) UpdateConfig(cfg *config.Config) error { if err := config.Save(cfg); err != nil { return err }; a.appConfig = cfg; return nil }

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
