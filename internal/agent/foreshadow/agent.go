package foreshadow

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"weaveforge/internal/llm"
	"weaveforge/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type chatClient interface {
	ChatCompletion(ctx context.Context, messages []llm.Message, model string, opts ...llm.ChatOption) (string, error)
}

type Agent struct {
	db        *gorm.DB
	llm       chatClient
	chatModel string
}

func NewAgent(db *gorm.DB, llm chatClient, chatModel string) *Agent {
	return &Agent{db: db, llm: llm, chatModel: chatModel}
}

// ─── Regex-based auto-detection patterns ──────────────────────────────

var foreshadowPatterns = []struct {
	pattern *regexp.Regexp
	ftype   string
}{
	{regexp.MustCompile(`(?i)(并不知道|还不清楚|未曾想到|从未听说过)`), "人物背景"},
	{regexp.MustCompile(`(?i)(后来才(明白|知道|发现|懂得))`), "时间谜团"},
	{regexp.MustCompile(`(?i)(小小的|看似寻常的|不起眼的|不起眼)`), "关键道具"},
	{regexp.MustCompile(`(?i)(将在(日后|未来|不久后|将来))`), "时间谜团"},
	{regexp.MustCompile(`(?i)(埋下了(种子|伏笔|隐患|祸根))`), "对话暗示"},
	{regexp.MustCompile(`(?i)(当时没有(注意|在意|发现|多想))`), "对话暗示"},
	{regexp.MustCompile(`(?i)(隐约觉得|隐隐约约|有种预感)`), "人物背景"},
	{regexp.MustCompile(`(?i)(若有所思|欲言又止|话到嘴边)`), "对话暗示"},
	{regexp.MustCompile(`(?i)(意味深长|别有深意)`), "对话暗示"},
}

// ─── Public API ────────────────────────────────────────────────────────

func (a *Agent) AutoDetectForeshadowing(ctx context.Context, chapterContent string) ([]CandidateForeshadow, error) {
	text := strings.TrimSpace(chapterContent)
	if text == "" {
		return nil, nil
	}

	// Step 1: Regex-based candidate extraction
	runes := []rune(text)
	byteToRune := buildByteToRuneMap(text)
	var regexCandidates []CandidateForeshadow
	for _, fp := range foreshadowPatterns {
		matches := fp.pattern.FindAllStringIndex(text, -1)
		for _, m := range matches {
			// Convert byte offsets to rune offsets
			rStart := byteToRune[m[0]]
			rEnd := byteToRune[m[1]]
			// Expand context: take ~40 chars around the match
			ctxStart := rStart - 15
			if ctxStart < 0 {
				ctxStart = 0
			}
			ctxEnd := rEnd + 25
			if ctxEnd > len(runes) {
				ctxEnd = len(runes)
			}
			snippet := string(runes[ctxStart:ctxEnd])
			regexCandidates = append(regexCandidates, CandidateForeshadow{
				Text:       snippet,
				Type:       fp.ftype,
				Confidence: 0.5,
				StartIndex: rStart,
				EndIndex:   rEnd,
			})
		}
	}
	if len(regexCandidates) > 10 {
		regexCandidates = regexCandidates[:10]
	}

	// Step 2: LLM verification & classification
	if len(regexCandidates) > 0 && a.llm != nil {
		var candidatesJSON strings.Builder
		for i, c := range regexCandidates {
			fmt.Fprintf(&candidatesJSON, `{"index":%d,"text":"%s"}`, i, strings.ReplaceAll(c.Text, `"`, `\"`))
			if i < len(regexCandidates)-1 {
				candidatesJSON.WriteString(",\n")
			}
		}

		prompt := fmt.Sprintf(`你是一位小说伏笔分析专家。

请分析以下候选文本，判断是否可能构成伏笔，并给出类型和置信度。

可能的伏笔类型：人物背景、关键道具、时间谜团、对话暗示

候选文本（JSON）：
[%s]

对每个候选，返回JSON数组（只输出JSON）：
[{"index":0,"is_foreshadow":true,"type":"人物背景","confidence":0.8,"reason":"暗示角色有隐藏过去"}]

如果没有伏笔，返回空数组[]。`, candidatesJSON.String())

		resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
			{Role: "system", Content: "你是一位小说伏笔分析专家。只输出JSON数组，不要其他文字。"},
			{Role: "user", Content: prompt},
		}, a.chatModel, llm.ChatOption{Temperature: 0.1})
		if err == nil {
			// Parse LLM response
			var verifications []struct {
				Index        int     `json:"index"`
				IsForeshadow bool    `json:"is_foreshadow"`
				Type         string  `json:"type"`
				Confidence   float64 `json:"confidence"`
				Reason       string  `json:"reason"`
			}
			start := strings.Index(resp, "[")
			end := strings.LastIndex(resp, "]")
			if start >= 0 && end > start {
				if jsonErr := json.Unmarshal([]byte(resp[start:end+1]), &verifications); jsonErr == nil {
					var result []CandidateForeshadow
					for _, v := range verifications {
						if v.IsForeshadow && v.Index < len(regexCandidates) {
							c := regexCandidates[v.Index]
							c.Type = v.Type
							c.Confidence = v.Confidence
							c.Reason = v.Reason
							if c.Confidence > 0.4 {
								result = append(result, c)
							}
						}
					}
					return result, nil
				}
			}
		}
	}

	// Fallback: return regex candidates with >0.5 confidence
	var result []CandidateForeshadow
	for _, c := range regexCandidates {
		if c.Confidence >= 0.5 {
			result = append(result, c)
		}
	}
	return result, nil
}

func (a *Agent) ConfirmForeshadowing(ctx context.Context, c CandidateForeshadow, chapterID string) (string, error) {
	id := uuid.New().String()
	f := models.Foreshadowing{
		ID:             id,
		Description:    c.Text,
		Type:           c.Type,
		Status:         "planted",
		ChapterPlanted: chapterID,
		Confidence:     c.Confidence,
	}
	if err := a.db.WithContext(ctx).Create(&f).Error; err != nil {
		return "", fmt.Errorf("foreshadow: save: %w", err)
	}
	return id, nil
}

func (a *Agent) SaveForeshadowing(ctx context.Context, text string, chapterID string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("foreshadow: empty text")
	}
	id := uuid.New().String()
	f := models.Foreshadowing{
		ID:             id,
		Description:    text,
		Type:           "手动标记",
		Status:         "planted",
		ChapterPlanted: chapterID,
		Confidence:     1.0,
	}
	if err := a.db.WithContext(ctx).Create(&f).Error; err != nil {
		return "", fmt.Errorf("foreshadow: save: %w", err)
	}
	return id, nil
}

func (a *Agent) ListForeshadowings(ctx context.Context, statusFilter string) ([]ForeshadowSummary, error) {
	var rows []models.Foreshadowing
	q := a.db.WithContext(ctx).Order("created_at desc")
	if statusFilter != "" && statusFilter != "all" {
		q = q.Where("status = ?", statusFilter)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]ForeshadowSummary, len(rows))
	for i, r := range rows {
		desc := r.Description
		if len([]rune(desc)) > 80 {
			desc = string([]rune(desc)[:80]) + "…"
		}
		result[i] = ForeshadowSummary{
			ID: r.ID, Description: desc, Type: r.Type, Status: r.Status,
			ChapterPlanted: r.ChapterPlanted, PlannedRevealChapter: r.PlannedRevealChapter,
			RevealProgress: r.RevealProgress, Priority: r.Priority, Confidence: r.Confidence,
			CreatedAt: r.CreatedAt.Format("2006-01-02 15:04"),
		}
	}
	return result, nil
}

func (a *Agent) GetForeshadowing(ctx context.Context, id string) (*models.Foreshadowing, error) {
	var f models.Foreshadowing
	if err := a.db.WithContext(ctx).First(&f, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (a *Agent) UpdateForeshadowing(ctx context.Context, id string, updates map[string]interface{}) error {
	allowed := map[string]bool{"status": true, "type": true, "priority": true,
		"planned_reveal_chapter": true, "reveal_progress": true, "description": true, "reveal_plan": true}
	filtered := make(map[string]interface{})
	for k, v := range updates {
		if allowed[k] {
			filtered[k] = v
		}
	}
	return a.db.WithContext(ctx).Model(&models.Foreshadowing{}).Where("id = ?", id).Updates(filtered).Error
}

func (a *Agent) DeleteForeshadowing(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Delete(&models.Foreshadowing{}, "id = ?", id).Error
}

// ─── Reveal Suggestions ────────────────────────────────────────────────

func (a *Agent) SuggestReveal(ctx context.Context, currentChapterIndex int) ([]RevealSuggestion, error) {
	var rows []models.Foreshadowing
	if err := a.db.WithContext(ctx).Where("status != ?", "revealed").
		Order("planned_reveal_chapter asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	// Filter: planned reveal near current chapter (~5 chapters range)
	var candidates []models.Foreshadowing
	for _, r := range rows {
		gap := math.Abs(float64(r.PlannedRevealChapter - currentChapterIndex))
		if gap <= 5 || r.PlannedRevealChapter == 0 {
			candidates = append(candidates, r)
		}
	}
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// LLM: generate reveal plans
	var itemsJSON strings.Builder
	for i, c := range candidates {
		desc := c.Description
		if len([]rune(desc)) > 100 {
			desc = string([]rune(desc)[:100])
		}
		fmt.Fprintf(&itemsJSON, `{"id":"%s","description":"%s","type":"%s"}`, c.ID, desc, c.Type)
		if i < len(candidates)-1 {
			itemsJSON.WriteString(",\n")
		}
	}

	prompt := fmt.Sprintf(`你是一位小说创作助手。以下伏笔需要在第 %d 章附近揭示。

伏笔列表：
[%s]

为每个伏笔生成 2-3 种揭示方案。方案类型："侧面揭示"、"直接回忆"、"事件触发"。
每种方案包含一段融入本章的草稿段落（150字以内）。

只输出JSON数组：`+`[{"id":"伏笔ID","plans":[{"method":"方案类型","paragraph":"草稿段落"}]}]`+`

如果没有合适的方案，返回空数组[]。`, currentChapterIndex, itemsJSON.String())

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是小说创作助手，为伏笔设计揭示方案。只输出JSON数组。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.6})
	if err != nil {
		return nil, fmt.Errorf("foreshadow: llm suggest: %w", err)
	}

	var plans []struct {
		ID    string       `json:"id"`
		Plans []RevealPlan `json:"plans"`
	}
	start := strings.Index(resp, "[")
	end := strings.LastIndex(resp, "]")
	if start < 0 || end <= start {
		return nil, nil
	}
	if json.Unmarshal([]byte(resp[start:end+1]), &plans) != nil {
		return nil, nil
	}

	// Match back to candidates
	results := make([]RevealSuggestion, 0, len(plans))
	for _, p := range plans {
		for _, c := range candidates {
			if c.ID == p.ID {
				results = append(results, RevealSuggestion{
					ForeshadowID:   c.ID,
					Description:    c.Description,
					ForeshadowType: c.Type,
					CurrentStatus:  c.Status,
					Plans:          p.Plans,
				})
				break
			}
		}
	}
	return results, nil
}

// buildByteToRuneMap builds a mapping from byte offset to rune offset for a string.
// This is needed because regexp.FindAllStringIndex returns byte offsets,
// but we work with rune slices for correct Unicode handling.
func buildByteToRuneMap(s string) []int {
	// byteToRune[bytePos] = runePos
	m := make([]int, len(s)+1)
	runePos := 0
	for i := range s {
		m[i] = runePos
		runePos++
	}
	m[len(s)] = runePos
	return m
}

// ─── Health Report ─────────────────────────────────────────────────────

func (a *Agent) GenerateHealthReport(ctx context.Context) (*HealthReport, error) {
	var all []models.Foreshadowing
	if err := a.db.WithContext(ctx).Find(&all).Error; err != nil {
		return nil, err
	}

	r := &HealthReport{
		Total:          len(all),
		PriorityCounts: map[string]int{},
	}

	var revealGaps []float64
	for _, f := range all {
		r.PriorityCounts[f.Priority]++
		switch f.Status {
		case "planted":
			r.Planted++
		case "partially_revealed":
			r.PartiallyRevealed++
		case "revealed":
			r.Revealed++
		}
		if f.PlannedRevealChapter > 0 {
			// Approximate: chapter_planted might be a UUID, use the numeric sort order
			gap := float64(f.PlannedRevealChapter)
			revealGaps = append(revealGaps, gap)
		}
		// Check stale: planted with no progress for >30 days
		if f.Status == "planted" || f.Status == "partially_revealed" {
			daysSince := time.Since(f.UpdatedAt).Hours() / 24
			if daysSince > 30 {
				r.StaleCount++
				desc := f.Description
				if len([]rune(desc)) > 40 {
					desc = string([]rune(desc)[:40]) + "…"
				}
				r.StaleWarnings = append(r.StaleWarnings,
					fmt.Sprintf("「%s」已 %.0f 天无进展", desc, daysSince))
			}
		}
	}

	if len(revealGaps) > 0 {
		var sum float64
		for _, g := range revealGaps {
			sum += g
		}
		r.AvgRevealGap = math.Round(sum/float64(len(revealGaps))*10) / 10
	}

	return r, nil
}
