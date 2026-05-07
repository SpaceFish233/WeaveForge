package setting

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"weaveforge/internal/llm"
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
	if err := a.store.DeleteDocuments(ctx, map[string]string{"setting_id": id}); err != nil {
		return err
	}
	if err := a.db.WithContext(ctx).Model(&models.WorldSetting{}).Where("id = ?", id).
		Updates(map[string]interface{}{"title": title, "content": content, "type": settingType}).Error; err != nil {
		return err
	}
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
	return a.store.AddDocuments(ctx, docs)
}

func (a *Agent) DeleteSetting(ctx context.Context, id string) error {
	if err := a.store.DeleteDocuments(ctx, map[string]string{"setting_id": id}); err != nil {
		return err
	}
	return a.db.WithContext(ctx).Delete(&models.WorldSetting{}, "id = ?", id).Error
}

func (a *Agent) CheckConsistency(ctx context.Context, chapterContent string) ([]ConflictWarning, error) {
	if strings.TrimSpace(chapterContent) == "" {
		return nil, nil
	}
	results, err := a.store.SearchSimilar(ctx, chapterContent, 5)
	if err != nil {
		return nil, fmt.Errorf("setting: search: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	var sb strings.Builder
	sb.WriteString("以下是该作品的设定资料：\n\n")
	for _, r := range results {
		var meta map[string]string
		json.Unmarshal([]byte(r.Metadata), &meta)
		title := meta["title"]
		if title == "" {
			title = "未命名设定"
		}
		sb.WriteString(fmt.Sprintf("【%s】\n%s\n\n", title, r.Content))
	}
	sb.WriteString("---\n待检查的小说段落：\n" + chapterContent)

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是一个网络小说设定一致性检查专家。检查小说内容是否与设定资料存在矛盾。对每个冲突以JSON数组格式返回，每个元素包含 conflict_desc（冲突描述）、suggested_fix（建议修改）、reference_text（引用设定原文）。如果没有冲突返回空数组 []。只输出JSON，不要其他文字。"},
		{Role: "user", Content: sb.String()},
	}, a.chatModel, llm.ChatOption{Temperature: 0.1})
	if err != nil {
		return nil, fmt.Errorf("setting: llm: %w", err)
	}
	return parseLLMResponse(resp, results)
}

func parseLLMResponse(resp string, refDocs []vectordb.Document) ([]ConflictWarning, error) {
	re := regexp.MustCompile(`\[[\s\S]*\]`)
	matches := re.FindString(resp)
	if matches == "" {
		return nil, fmt.Errorf("no JSON array in LLM response")
	}
	var jc []jsonConflict
	if err := json.Unmarshal([]byte(matches), &jc); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	refMap := make(map[string]string)
	for _, d := range refDocs {
		var meta map[string]string
		if err := json.Unmarshal([]byte(d.Metadata), &meta); err == nil {
			if t := meta["title"]; t != "" {
				refMap[d.Content[:len(d.Content)/2]] = t
			}
		}
	}
	warnings := make([]ConflictWarning, 0, len(jc))
	for _, c := range jc {
		if c.ConflictDesc == "" {
			continue
		}
		title := ""
		for ref, t := range refMap {
			if strings.Contains(c.ReferenceText, ref) || strings.Contains(ref, c.ReferenceText) {
				title = t
				break
			}
		}
		warnings = append(warnings, ConflictWarning{ConflictDesc: c.ConflictDesc, SuggestedFix: c.SuggestedFix, ReferenceText: c.ReferenceText, SettingTitle: title})
	}
	return warnings, nil
}

// --- chunkText / splitSentences (same as before) ---

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
