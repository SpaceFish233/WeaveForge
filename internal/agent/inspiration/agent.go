package inspiration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	db        *gorm.DB
	store     vectordb.VectorStore
	embed     vectordb.Embedder
	llm       chatClient
	chatModel string
}

func NewAgent(db *gorm.DB, store vectordb.VectorStore, embed vectordb.Embedder, llm chatClient, chatModel string) *Agent {
	return &Agent{db: db, store: store, embed: embed, llm: llm, chatModel: chatModel}
}

func (a *Agent) SaveInspiration(ctx context.Context, content string, tags []string) error {
	id := uuid.New().String()
	tagsJSON, _ := json.Marshal(tags)
	insp := models.Inspiration{ID: id, Content: content, Tags: string(tagsJSON), Source: "manual"}
	if err := a.db.WithContext(ctx).Create(&insp).Error; err != nil {
		return fmt.Errorf("inspiration: save: %w", err)
	}
	emb, err := a.embed.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("inspiration: embed: %w", err)
	}
	meta, _ := json.Marshal(map[string]string{"source": "inspiration", "inspiration_id": id})
	return a.store.AddDocuments(ctx, []vectordb.Document{{
		ID: uuid.New().String(), Content: content, Metadata: string(meta), Embedding: emb,
	}})
}

func (a *Agent) ListInspirations(ctx context.Context, filterTags []string) ([]InspirationSummary, error) {
	var rows []models.Inspiration
	q := a.db.WithContext(ctx).Order("created_at desc")
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(filterTags) > 0 {
		var filtered []models.Inspiration
		for _, r := range rows {
			var tags []string
			json.Unmarshal([]byte(r.Tags), &tags)
			if tagsMatchAny(tags, filterTags) {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}
	result := make([]InspirationSummary, len(rows))
	for i, r := range rows {
		var tags []string
		json.Unmarshal([]byte(r.Tags), &tags)
		content := r.Content
		if len([]rune(content)) > 100 {
			content = string([]rune(content)[:100]) + "…"
		}
		result[i] = InspirationSummary{ID: r.ID, Content: content, Tags: tags, Source: r.Source, CreatedAt: r.CreatedAt.Format("2006-01-02 15:04")}
	}
	return result, nil
}

func (a *Agent) DeleteInspiration(ctx context.Context, id string) error {
	if err := a.store.DeleteDocuments(ctx, map[string]string{"inspiration_id": id}); err != nil {
		return err
	}
	return a.db.WithContext(ctx).Delete(&models.Inspiration{}, "id = ?", id).Error
}

func (a *Agent) ContextPush(ctx context.Context, sceneType string, keywords []string) ([]InspirationMatch, error) {
	var parts []string
	if sceneType != "" {
		parts = append(parts, sceneType)
	}
	parts = append(parts, keywords...)
	query := strings.Join(parts, " ")
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	filtered, err := a.store.SearchSimilarFilter(ctx, query, 3, map[string]string{"source": "inspiration"})
	if err != nil {
		return nil, fmt.Errorf("inspiration: search: %w", err)
	}
	matches := make([]InspirationMatch, 0, len(filtered))
	for _, d := range filtered {
		var meta map[string]string
		json.Unmarshal([]byte(d.Metadata), &meta)
		inspID := meta["inspiration_id"]
		var tags []string
		if inspID != "" {
			var insp models.Inspiration
			if err := a.db.WithContext(ctx).First(&insp, "id = ?", inspID).Error; err == nil {
				json.Unmarshal([]byte(insp.Tags), &tags)
			}
		}
		matches = append(matches, InspirationMatch{
			ID: inspID, Content: d.Content, Tags: tags, Score: d.Score, MatchType: "semantic",
		})
	}
	return matches, nil
}

func (a *Agent) MarkAsDeprecated(ctx context.Context, chapterID, segmentContent string) ([]string, error) {
	if strings.TrimSpace(segmentContent) == "" {
		return nil, fmt.Errorf("inspiration: empty content")
	}
	chapterTitle := ""
	if chapterID != "" {
		var ch models.Chapter
		if err := a.db.WithContext(ctx).First(&ch, "id = ?", chapterID).Error; err == nil {
			chapterTitle = ch.Title
		}
	}
	var sb strings.Builder
	sb.WriteString("你是一位小说创作助手。以下是一段被作者弃用的文本片段，请从中提取有价值的创作素材。\n\n")
	if chapterTitle != "" {
		sb.WriteString(fmt.Sprintf("原文所在章节：%s\n\n", chapterTitle))
	}
	sb.WriteString("【弃用文本】\n" + segmentContent + "\n\n")
	sb.WriteString("请将这段文本拆解为以下元素的JSON数组：\n")
	sb.WriteString(`[{"type":"场景","content":"场景描述文本"},{"type":"对白","content":"对白文本"},{"type":"设定碎片","content":"设定元素"}]` + "\n")
	sb.WriteString("如果某个类型没有找到对应内容，就不要包含该条目。只输出JSON数组，不要其他文字。")

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是一个文本分析助手，从小说文本中提取场景、对白和设定元素，只输出JSON。"},
		{Role: "user", Content: sb.String()},
	}, a.chatModel, llm.ChatOption{Temperature: 0.1})
	if err != nil {
		return nil, fmt.Errorf("inspiration: llm analyze: %w", err)
	}

	var elements []struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	start := strings.Index(resp, "[")
	end := strings.LastIndex(resp, "]")
	if start >= 0 && end > start {
		resp = resp[start : end+1]
	}
	if err := json.Unmarshal([]byte(resp), &elements); err != nil {
		elements = []struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}{{Type: "设定碎片", Content: segmentContent}}
	}

	var createdIDs []string
	for _, el := range elements {
		if strings.TrimSpace(el.Content) == "" {
			continue
		}
		id := uuid.New().String()
		tagsJSON, _ := json.Marshal([]string{"弃用", el.Type})
		insp := models.Inspiration{ID: id, Content: el.Content, Tags: string(tagsJSON), Source: "deprecated", SourceRef: chapterID}
		if err := a.db.WithContext(ctx).Create(&insp).Error; err != nil {
			continue
		}
		emb, err := a.embed.Embed(ctx, el.Content)
		if err != nil {
			continue
		}
		meta, _ := json.Marshal(map[string]string{"source": "inspiration", "inspiration_id": id, "insp_type": el.Type})
		a.store.AddDocuments(ctx, []vectordb.Document{{ID: uuid.New().String(), Content: el.Content, Metadata: string(meta), Embedding: emb}})
		createdIDs = append(createdIDs, id)
	}
	return createdIDs, nil
}

func tagsMatchAny(tags, filter []string) bool {
	for _, t := range tags {
		for _, f := range filter {
			if strings.EqualFold(t, f) {
				return true
			}
		}
	}
	return false
}
