package main

import (
	"context"
	"strings"
	"weaveforge/internal/agent/character"
	"weaveforge/internal/agent/foreshadow"
	"weaveforge/internal/agent/inspiration"
	"weaveforge/internal/agent/plotengine"
	"weaveforge/internal/agent/setting"
	"weaveforge/internal/agent/style"
	"weaveforge/internal/config"
	"weaveforge/internal/coordinator"
	"weaveforge/internal/llm"
	"weaveforge/models"
	"weaveforge/parser"
	"weaveforge/services"
)

type App struct {
	ctx              context.Context
	chapter          *services.ChapterService
	characterAgent   *character.Agent
	settingAgent     *setting.Agent
	styleAgent       *style.Agent
	inspirationAgent *inspiration.Agent
	foreshadowAgent  *foreshadow.Agent
	plotEngine       *plotengine.Agent
	coordinator      *coordinator.Coordinator
	appConfig        *config.Config
}

func NewApp(cs *services.ChapterService, ca *character.Agent, sa *setting.Agent, sta *style.Agent, ia *inspiration.Agent, fa *foreshadow.Agent, pe *plotengine.Agent, co *coordinator.Coordinator, cfg *config.Config) *App {
	return &App{chapter: cs, characterAgent: ca, settingAgent: sa, styleAgent: sta, inspirationAgent: ia, foreshadowAgent: fa, plotEngine: pe, coordinator: co, appConfig: cfg}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// ─── Chapter API ────────────────────────────────────────────────────

func (a *App) CreateChapter(title, content string) (string, error)     { return a.chapter.CreateChapter(title, content) }
func (a *App) UpdateChapter(chapterID, content string) error           { return a.chapter.UpdateChapter(chapterID, content) }
func (a *App) UpdateChapterTitle(chapterID, title string) error        { return a.chapter.UpdateChapterTitle(chapterID, title) }
func (a *App) DeleteChapter(chapterID string) error                    { return a.chapter.DeleteChapter(chapterID) }
func (a *App) GetChapter(chapterID string) (models.Chapter, error)     { return a.chapter.GetChapter(chapterID) }
func (a *App) ListChapters() ([]models.ChapterSummary, error)          { return a.chapter.ListChapters() }
func (a *App) ReorderChapters(chapterIDs []string) error               { return a.chapter.ReorderChapters(chapterIDs) }

func (a *App) ImportDocument(filePath string) ([]string, error) { chs, err := parser.ParseFile(filePath); if err != nil { return nil, err }; return a.importChapters(chs) }
func (a *App) ImportContent(filename, content string) ([]string, error) { ext := ""; if idx := strings.LastIndex(filename, "."); idx >= 0 { ext = filename[idx:] }; return a.importChapters(parser.ParseContent(content, ext)) }
func (a *App) importChapters(chs []parser.ParsedChapter) ([]string, error) { var ids []string; for _, ch := range chs { id, err := a.chapter.CreateChapter(ch.Title, ch.Content); if err != nil { return ids, err }; ids = append(ids, id) }; return ids, nil }

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
func (a *App) CheckConsistency(chapterContent string) ([]setting.ConflictWarning, error) { return a.settingAgent.CheckConsistency(a.ctx, chapterContent) }

// ─── Style Agent API ────────────────────────────────────────────────

func (a *App) LearnStyle(name string, chapterIDs []string) (string, error)        { return a.styleAgent.LearnStyle(a.ctx, name, chapterIDs) }
func (a *App) ListStyleProfiles() ([]style.StyleProfileSummary, error)            { return a.styleAgent.ListProfiles(a.ctx) }
func (a *App) GetStyleProfile(id string) (*models.StyleProfile, error)            { return a.styleAgent.GetProfile(a.ctx, id) }
func (a *App) DeleteStyleProfile(id string) error                                  { return a.styleAgent.DeleteProfile(a.ctx, id) }
func (a *App) PolishText(text, profileID, intensity string) (string, error)       { return a.styleAgent.PolishText(a.ctx, text, profileID, intensity) }
func (a *App) AnalyzeStyle(text, profileID string) (string, error)                { return a.styleAgent.AnalyzeStyle(a.ctx, text, profileID) }

// ─── Inspiration Agent API ──────────────────────────────────────────

func (a *App) SaveInspiration(content string, tags []string) error                          { return a.inspirationAgent.SaveInspiration(a.ctx, content, tags) }
func (a *App) ListInspirations(filterTags []string) ([]inspiration.InspirationSummary, error) { return a.inspirationAgent.ListInspirations(a.ctx, filterTags) }
func (a *App) DeleteInspiration(id string) error                                              { return a.inspirationAgent.DeleteInspiration(a.ctx, id) }
func (a *App) ContextPush(sceneType string, keywords []string) ([]inspiration.InspirationMatch, error) { return a.inspirationAgent.ContextPush(a.ctx, sceneType, keywords) }
func (a *App) MarkAsDeprecated(chapterID string, segmentContent string) ([]string, error)   { return a.inspirationAgent.MarkAsDeprecated(a.ctx, chapterID, segmentContent) }

// ─── Foreshadow Agent API ───────────────────────────────────────────

func (a *App) AutoDetectForeshadowing(chapterContent string) ([]foreshadow.CandidateForeshadow, error) { return a.foreshadowAgent.AutoDetectForeshadowing(a.ctx, chapterContent) }
func (a *App) ConfirmForeshadowing(candidate foreshadow.CandidateForeshadow, chapterID string) (string, error) { id, err := a.foreshadowAgent.ConfirmForeshadowing(a.ctx, candidate, chapterID); if err == nil { go a.settingAgent.CheckConsistency(context.Background(), candidate.Text) }; return id, err }
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

// ─── Coordinator API ───────────────────────────────────────────────

func (a *App) OnParagraphWritten(chapterID, paragraphText string) { a.coordinator.OnParagraphWritten(chapterID, paragraphText) }
func (a *App) SetAssistantIntensity(level int) { a.coordinator.SetIntensity(level) }
func (a *App) GetAssistantIntensity() int { return a.coordinator.GetIntensity() }
func (a *App) GetSessionHistory() []coordinator.SessionEvent { return a.coordinator.GetSessionHistory() }
func (a *App) RecordNotificationAction(notifID, action string) { a.coordinator.RecordUserAction(notifID, action) }
