package setting

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode/utf8"

	"weaveforge/internal/llm"
	"weaveforge/internal/textutil"
	"weaveforge/internal/vectordb"
	"weaveforge/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type chatClient interface {
	ChatCompletion(ctx context.Context, messages []llm.Message, model string, opts ...llm.ChatOption) (string, error)
}

type Agent struct {
	store     vectordb.VectorStore
	embed     vectordb.Embedder
	llm       chatClient
	chatModel string
	db        *gorm.DB
}

func NewAgent(store vectordb.VectorStore, embed vectordb.Embedder, llm chatClient, chatModel string, db *gorm.DB) *Agent {
	return &Agent{store: store, embed: embed, llm: llm, chatModel: chatModel, db: db}
}

func (a *Agent) UploadWorldSetting(ctx context.Context, title, content, settingType string) error {
	id := uuid.New().String()
	if err := a.db.WithContext(ctx).Create(&models.WorldSetting{
		ID: id, Title: title, Content: content, Type: settingType,
	}).Error; err != nil {
		return fmt.Errorf("setting: save: %w", err)
	}
	chunks := chunkText(content, 500)
	if len(chunks) == 0 {
		return nil
	}
	var docs []vectordb.Document
	for i, chunk := range chunks {
		emb, err := a.embed.Embed(ctx, chunk)
		if err != nil {
			return fmt.Errorf("setting: embed chunk %d: %w", i, err)
		}
		meta, _ := json.Marshal(map[string]string{
			"title": title, "type": settingType, "setting_id": id, "chunk_index": fmt.Sprintf("%d", i),
		})
		docs = append(docs, vectordb.Document{
			ID: uuid.New().String(), Content: chunk, Metadata: string(meta), Embedding: emb,
		})
	}
	return a.store.AddDocuments(ctx, docs)
}

func (a *Agent) ListSettings(ctx context.Context) ([]SettingInfo, error) {
	var settings []models.WorldSetting
	if err := a.db.WithContext(ctx).Order("type asc, title asc").Find(&settings).Error; err != nil {
		return nil, err
	}
	infos := make([]SettingInfo, len(settings))
	for i, s := range settings {
		infos[i] = SettingInfo{ID: s.ID, Title: s.Title, Type: s.Type}
	}
	return infos, nil
}

func (a *Agent) GetSetting(ctx context.Context, id string) (*models.WorldSetting, error) {
	var s models.WorldSetting
	if err := a.db.WithContext(ctx).First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (a *Agent) UpdateSetting(ctx context.Context, id, title, content, settingType string) error {
	// Generate new chunks and embeddings first
	chunks := chunkText(content, 500)
	var docs []vectordb.Document
	for i, chunk := range chunks {
		emb, err := a.embed.Embed(ctx, chunk)
		if err != nil {
			return fmt.Errorf("setting: re-embed chunk %d: %w", i, err)
		}
		meta, _ := json.Marshal(map[string]string{
			"title": title, "type": settingType, "setting_id": id, "chunk_index": fmt.Sprintf("%d", i),
		})
		docs = append(docs, vectordb.Document{
			ID: uuid.New().String(), Content: chunk, Metadata: string(meta), Embedding: emb,
		})
	}

	// Update vector store first: add new docs before deleting old ones so at
	// least one copy always exists. Vector ops are best-effort; if old doc
	// deletion fails, duplicates may remain but search results are unaffected.
	if len(docs) > 0 {
		if err := a.store.AddDocuments(ctx, docs); err != nil {
			return fmt.Errorf("setting: add new docs: %w", err)
		}
	}
	if err := a.store.DeleteDocuments(ctx, map[string]string{"setting_id": id}); err != nil {
		log.Printf("setting: delete old vector docs for %s: %v (non-fatal)", id, err)
	}

	// Update the database record last; if vector ops fail above, DB stays unchanged.
	if err := a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&models.WorldSetting{}).Where("id = ?", id).
			Updates(map[string]interface{}{"title": title, "content": content, "type": settingType}).Error
	}); err != nil {
		return err
	}
	return nil
}

func (a *Agent) DeleteSetting(ctx context.Context, id string) error {
	if err := a.store.DeleteDocuments(ctx, map[string]string{"setting_id": id}); err != nil {
		return err
	}
	return a.db.WithContext(ctx).Delete(&models.WorldSetting{}, "id = ?", id).Error
}

// ─── Detection (no AI) ────────────────────────────────────────────────

var isPunctOrSpace = regexp.MustCompile(`^[\p{P}\p{Z}\p{S}]+$`).MatchString

func (a *Agent) DetectSettingsInChapter(chapterContent string, caseSensitive bool, wholeWord bool) ([]SettingHit, error) {
	trimmed := strings.TrimSpace(chapterContent)
	if trimmed == "" {
		return nil, nil
	}

	var settings []models.WorldSetting
	if err := a.db.Order("type asc, title asc").Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("setting: list: %w", err)
	}

	var hits []SettingHit
	for _, s := range settings {
		title := strings.TrimSpace(s.Title)
		if title == "" || isPunctOrSpace(title) {
			continue
		}

		occurrences, positions := countOccurrences(chapterContent, title, caseSensitive, wholeWord)
		if occurrences == 0 {
			continue
		}

		snippet := extractSnippet(chapterContent, positions[0], title, 100)
		hits = append(hits, SettingHit{
			ID:           s.ID,
			Title:        title,
			Occurrences:  occurrences,
			FirstSnippet: snippet,
		})
	}
	return hits, nil
}

func countOccurrences(text, keyword string, caseSensitive, wholeWord bool) (int, []int) {
	src := text
	kw := keyword
	if !caseSensitive {
		src = strings.ToLower(text)
		kw = strings.ToLower(keyword)
	}

	if wholeWord {
		escaped := regexp.QuoteMeta(kw)
		re, err := regexp.Compile(`\b` + escaped + `\b`)
		if err != nil {
			return 0, nil
		}
		matches := re.FindAllStringIndex(src, -1)
		byteToRune := textutil.ByteToRuneMap(src)
		positions := make([]int, len(matches))
		for i, m := range matches {
			positions[i] = byteToRune[m[0]]
		}
		return len(matches), positions
	}

	byteToRune := textutil.ByteToRuneMap(src)
	var positions []int
	offset := 0
	for {
		idx := strings.Index(src[offset:], kw)
		if idx < 0 {
			break
		}
		positions = append(positions, byteToRune[offset+idx])
		offset += idx + len(kw)
	}
	return len(positions), positions
}

func extractSnippet(text string, runePos int, keyword string, contextWidth int) string {
	runes := []rune(text)
	kwLen := utf8.RuneCountInString(keyword)

	start := runePos - contextWidth
	if start < 0 {
		start = 0
	}
	end := runePos + kwLen + contextWidth
	if end > len(runes) {
		end = len(runes)
	}
	if start > len(runes) {
		start = len(runes)
	}
	if end > len(runes) {
		end = len(runes)
	}
	snippet := string(runes[start:end])
	if start > 0 {
		snippet = "…" + snippet
	}
	if end < len(runes) {
		snippet = snippet + "…"
	}
	return snippet
}

// ─── Validation (AI) ──────────────────────────────────────────────────

const maxSnippets = 20
const contextChars = 200

func (a *Agent) ValidateSettingConflict(ctx context.Context, settingID string, chapterContent string) (*ConflictResult, error) {
	s, err := a.GetSetting(ctx, settingID)
	if err != nil {
		return nil, fmt.Errorf("setting: get: %w", err)
	}

	_, positions := countOccurrences(chapterContent, s.Title, true, false)
	if len(positions) == 0 {
		return &ConflictResult{
			SettingTitle: s.Title,
			HasConflict:  false,
			ConflictDesc: "设定标题在正文中未找到（可能已被修改），请重新检测。",
		}, nil
	}

	truncated := 0
	if len(positions) > maxSnippets {
		truncated = len(positions) - maxSnippets
		positions = positions[:maxSnippets]
	}

	var snippets []string
	for _, pos := range positions {
		snippet := extractSnippet(chapterContent, pos, s.Title, contextChars)
		snippets = append(snippets, snippet)
	}

	prompt := buildValidationPrompt(s.Title, s.Content, snippets)
	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是一个网络小说设定一致性检查专家。检查小说段落是否与设定存在冲突。返回JSON格式结果，包含 has_conflict(布尔)、conflict_desc(冲突描述，无不填)、suggested_fix(修改建议，无不填)、reference_text(引用相关设定原文，无不填)。只输出JSON，不要其他文字。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.1})
	if err != nil {
		return nil, fmt.Errorf("setting: llm: %w", err)
	}

	result := parseValidationResponse(resp, s.Title, snippets, truncated)
	return result, nil
}

func buildValidationPrompt(title, content string, snippets []string) string {
	var sb strings.Builder
	sb.WriteString("【设定标题】")
	sb.WriteString(title)
	sb.WriteString("\n【设定描述】\n")
	sb.WriteString(content)
	sb.WriteString("\n\n【待检查的小说段落】\n")
	for i, sn := range snippets {
		sb.WriteString(fmt.Sprintf("--- 段落 %d ---\n%s\n\n", i+1, sn))
	}
	sb.WriteString("请逐段分析是否与设定存在冲突。")
	return sb.String()
}

func parseValidationResponse(resp, title string, snippets []string, truncated int) *ConflictResult {
	re := regexp.MustCompile(`\{[\s\S]*\}`)
	match := re.FindString(resp)
	if match == "" {
		return &ConflictResult{
			SettingTitle:  title,
			HasConflict:   false,
			ConflictDesc:  "无法解析 AI 返回结果",
			Snippets:      snippets,
			TruncatedFrom: truncated,
		}
	}

	var jc jsonConflict
	if err := json.Unmarshal([]byte(match), &jc); err != nil {
		return &ConflictResult{
			SettingTitle:  title,
			HasConflict:   false,
			ConflictDesc:  fmt.Sprintf("解析 AI 结果失败: %v", err),
			Snippets:      snippets,
			TruncatedFrom: truncated,
		}
	}

	return &ConflictResult{
		SettingTitle:  title,
		HasConflict:   jc.HasConflict,
		ConflictDesc:  jc.ConflictDesc,
		SuggestedFix:  jc.SuggestedFix,
		ReferenceText: jc.ReferenceText,
		Snippets:      snippets,
		TruncatedFrom: truncated,
	}
}

// ─── chunkText / splitSentences ───────────────────────────────────────

func chunkText(text string, maxChars int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= maxChars {
		return []string{text}
	}
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		paraRunes := utf8.RuneCountInString(para)
		if paraRunes > maxChars {
			flush()
			sentences := splitSentences(para)
			var seg strings.Builder
			for _, s := range sentences {
				sLen := utf8.RuneCountInString(s)
				if seg.Len() > 0 && seg.Len()+sLen > maxChars {
					chunks = append(chunks, strings.TrimSpace(seg.String()))
					seg.Reset()
				}
				seg.WriteString(s)
			}
			if seg.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(seg.String()))
			}
			continue
		}
		if current.Len() > 0 && current.Len()+paraRunes+2 > maxChars {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(para)
	}
	flush()
	return chunks
}

func splitSentences(text string) []string {
	var sentences []string
	var buf strings.Builder
	for _, r := range text {
		buf.WriteRune(r)
		if r == '。' || r == '！' || r == '？' || r == '\n' {
			sentences = append(sentences, buf.String())
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		sentences = append(sentences, buf.String())
	}
	return sentences
}
