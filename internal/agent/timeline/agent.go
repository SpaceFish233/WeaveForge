package timeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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

// ─── TimeBase CRUD ──────────────────────────────────────────────────

func (a *Agent) GetTimeBase(ctx context.Context) (*models.TimeBase, error) {
	var tb models.TimeBase
	if err := a.db.WithContext(ctx).First(&tb).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &tb, nil
}

func (a *Agent) SaveTimeBase(ctx context.Context, description, unit string) (*models.TimeBase, error) {
	var tb models.TimeBase
	err := a.db.WithContext(ctx).First(&tb).Error
	if err == gorm.ErrRecordNotFound {
		tb = models.TimeBase{
			ID:          uuid.New().String(),
			Description: description,
			Unit:        unit,
		}
		if err := a.db.WithContext(ctx).Create(&tb).Error; err != nil {
			return nil, err
		}
		return &tb, nil
	}
	if err != nil {
		return nil, err
	}
	tb.Description = description
	tb.Unit = unit
	if err := a.db.WithContext(ctx).Save(&tb).Error; err != nil {
		return nil, err
	}
	return &tb, nil
}

// ─── Node CRUD ──────────────────────────────────────────────────────

func (a *Agent) ListNodes(ctx context.Context) ([]NodeWithEvents, error) {
	var nodes []models.TimelineNode
	if err := a.db.WithContext(ctx).Order("offset_days asc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	result := make([]NodeWithEvents, len(nodes))
	for i, n := range nodes {
		events, _ := a.listEventsByNode(ctx, n.ID)
		result[i] = NodeWithEvents{
			ID:          n.ID,
			OffsetDays:  n.OffsetDays,
			Label:       n.Label,
			Description: n.Description,
			Events:      events,
		}
	}
	return result, nil
}

func (a *Agent) GetNode(ctx context.Context, id string) (*NodeWithEvents, error) {
	var n models.TimelineNode
	if err := a.db.WithContext(ctx).First(&n, "id = ?", id).Error; err != nil {
		return nil, err
	}
	events, _ := a.listEventsByNode(ctx, n.ID)
	return &NodeWithEvents{
		ID:          n.ID,
		OffsetDays:  n.OffsetDays,
		Label:       n.Label,
		Description: n.Description,
		Events:      events,
	}, nil
}

func (a *Agent) CreateNode(ctx context.Context, offsetDays float64, label, description string) (string, error) {
	n := models.TimelineNode{
		ID:          uuid.New().String(),
		OffsetDays:  offsetDays,
		Label:       label,
		Description: description,
	}
	if err := a.db.WithContext(ctx).Create(&n).Error; err != nil {
		return "", err
	}
	return n.ID, nil
}

func (a *Agent) UpdateNode(ctx context.Context, id string, offsetDays float64, label, description string) error {
	return a.db.WithContext(ctx).Model(&models.TimelineNode{}).Where("id = ?", id).Updates(map[string]interface{}{
		"offset_days": offsetDays,
		"label":       label,
		"description": description,
	}).Error
}

func (a *Agent) DeleteNode(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("node_id = ?", id).Delete(&models.TimelineEvent{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.TimelineNode{}, "id = ?", id).Error
	})
}

// ─── Event CRUD ─────────────────────────────────────────────────────

func (a *Agent) listEventsByNode(ctx context.Context, nodeID string) ([]EventSummary, error) {
	var events []models.TimelineEvent
	if err := a.db.WithContext(ctx).Where("node_id = ?", nodeID).Order("sort_order asc").Find(&events).Error; err != nil {
		return nil, err
	}
	result := make([]EventSummary, len(events))
	for i, e := range events {
		var cids []string
		json.Unmarshal([]byte(e.ChapterIDs), &cids)
		result[i] = EventSummary{
			ID:          e.ID,
			NodeID:      e.NodeID,
			Title:       e.Title,
			Summary:     e.Summary,
			ChapterIDs:  cids,
			IsGradual:   e.IsGradual,
			EventName:   e.EventName,
			RawTimeExpr: e.RawTimeExpr,
			SortOrder:   e.SortOrder,
		}
	}
	return result, nil
}

func (a *Agent) CreateEvent(ctx context.Context, nodeID, title, summary string, chapterIDs []string, isGradual bool, eventName, rawTimeExpr string) (string, error) {
	cids, _ := json.Marshal(chapterIDs)
	e := models.TimelineEvent{
		ID:          uuid.New().String(),
		NodeID:      nodeID,
		Title:       title,
		Summary:     summary,
		ChapterIDs:  string(cids),
		IsGradual:   isGradual,
		EventName:   eventName,
		RawTimeExpr: rawTimeExpr,
	}
	if err := a.db.WithContext(ctx).Create(&e).Error; err != nil {
		return "", err
	}
	return e.ID, nil
}

func (a *Agent) UpdateEvent(ctx context.Context, id, title, summary string, chapterIDs []string, isGradual bool, eventName string) error {
	cids, _ := json.Marshal(chapterIDs)
	return a.db.WithContext(ctx).Model(&models.TimelineEvent{}).Where("id = ?", id).Updates(map[string]interface{}{
		"title":       title,
		"summary":     summary,
		"chapter_ids": string(cids),
		"is_gradual":  isGradual,
		"event_name":  eventName,
	}).Error
}

func (a *Agent) DeleteEvent(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Delete(&models.TimelineEvent{}, "id = ?", id).Error
}

// ─── Scan Chapters ──────────────────────────────────────────────────

func (a *Agent) ScanChapters(ctx context.Context, chapterIDs []string) (*ScanResult, error) {
	// Load chapters
	var chapters []models.Chapter
	if err := a.db.WithContext(ctx).Where("id IN ?", chapterIDs).Order("sort_order asc").Find(&chapters).Error; err != nil {
		return nil, fmt.Errorf("timeline: load chapters: %w", err)
	}
	if len(chapters) == 0 {
		return &ScanResult{}, nil
	}

	// Get time base for offset calculation
	tb, _ := a.GetTimeBase(ctx)

	// Extract time expressions from each chapter
	var allEvents []extractedEvent
	for _, ch := range chapters {
		events := a.extractFromChapter(ctx, ch)
		allEvents = append(allEvents, events...)
	}

	// Merge gradual revelation events
	merged := mergeExtractedEvents(allEvents)

	// Create or find nodes and attach events
	var nodes []NodeWithEvents
	for _, ev := range merged {
		nodeID := ev.NodeID
		if nodeID == "" {
			// Create new node
			offset := ev.Offset
			if tb == nil {
				offset = float64(len(nodes)) * 10 // fallback spacing
			}
			nid, err := a.CreateNode(ctx, offset, ev.Label, ev.Description)
			if err != nil {
				continue
			}
			nodeID = nid
		}

		// Create event
		_, err := a.CreateEvent(ctx, nodeID, ev.Title, ev.Summary, ev.ChapterIDs, ev.IsGradual, ev.EventName, ev.RawTimeExpr)
		if err != nil {
			continue
		}
	}

	// Return full timeline
	nodes, err := a.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	// Simple conflict detection
	conflicts := a.detectConflicts(ctx, nodes)

	return &ScanResult{
		Nodes:     nodes,
		Conflicts: conflicts,
	}, nil
}

type extractedEvent struct {
	NodeID      string
	Offset      float64
	Label       string
	Description string
	Title       string
	Summary     string
	ChapterIDs  []string
	IsGradual   bool
	EventName   string
	RawTimeExpr string
}

// mergeExtractedEvents groups events by EventName and merges gradual ones.
func mergeExtractedEvents(events []extractedEvent) []extractedEvent {
	grouped := make(map[string][]extractedEvent)
	for _, ev := range events {
		key := ev.EventName
		if key == "" {
			key = ev.Title + ev.RawTimeExpr
		}
		grouped[key] = append(grouped[key], ev)
	}

	var result []extractedEvent
	for _, group := range grouped {
		if len(group) == 1 {
			result = append(result, group[0])
			continue
		}
		// Merge: combine chapter IDs, mark as gradual
		base := group[0]
		chapterSet := make(map[string]bool)
		for _, g := range group {
			for _, cid := range g.ChapterIDs {
				chapterSet[cid] = true
			}
		}
		var cids []string
		for cid := range chapterSet {
			cids = append(cids, cid)
		}
		base.ChapterIDs = cids
		base.IsGradual = len(group) > 1
		result = append(result, base)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Offset < result[j].Offset
	})
	return result
}

// detectConflicts checks for obvious time conflicts.
func (a *Agent) detectConflicts(ctx context.Context, nodes []NodeWithEvents) []ConflictInfo {
	var conflicts []ConflictInfo
	// Simple check: if two events for the same character happen at nearly the same time
	// but in different chapters far apart, flag it.
	// For now, a basic implementation that checks sequential chapter ordering vs time ordering.
	chapterOrder := make(map[string]int)
	var allEvents []EventSummary
	for i, n := range nodes {
		for _, ev := range n.Events {
			allEvents = append(allEvents, ev)
			for _, cid := range ev.ChapterIDs {
				if _, exists := chapterOrder[cid]; !exists {
					chapterOrder[cid] = i
				}
			}
		}
	}

	// Check if events from later chapters appear earlier on the timeline
	for i, ev1 := range allEvents {
		for _, cid1 := range ev1.ChapterIDs {
			for _, ev2 := range allEvents[i+1:] {
				for _, cid2 := range ev2.ChapterIDs {
					o1, ok1 := chapterOrder[cid1]
					o2, ok2 := chapterOrder[cid2]
					if ok1 && ok2 && o1 > o2 && ev1.Title != ev2.Title {
						// Later chapter event appears earlier on timeline
						conflicts = append(conflicts, ConflictInfo{
							Description: fmt.Sprintf("「%s」(章节较后)在时间线上早于「%s」(章节较前)", ev1.Title, ev2.Title),
							ChapterIDs:  []string{cid1, cid2},
							EventTitles: []string{ev1.Title, ev2.Title},
						})
						if len(conflicts) >= 5 {
							return conflicts
						}
					}
				}
			}
		}
	}
	return conflicts
}

// ─── LLM-based extraction ──────────────────────────────────────────

func (a *Agent) extractFromChapter(ctx context.Context, ch models.Chapter) []extractedEvent {
	// First, try regex extraction
	regexEvents := extractByRegex(ch.Content, ch.ID)
	// Then use LLM for richer extraction
	llmEvents := a.extractByLLM(ctx, ch.Content, ch.ID)

	// Deduplicate: prefer LLM results, skip regex events that overlap
	// Two events are duplicates if they share the same RawTimeExpr or Title.
	seen := make(map[string]bool)
	var all []extractedEvent
	for _, ev := range llmEvents {
		key := strings.ToLower(strings.TrimSpace(ev.RawTimeExpr + "|" + ev.Title))
		seen[key] = true
		all = append(all, ev)
	}
	for _, ev := range regexEvents {
		key := strings.ToLower(strings.TrimSpace(ev.RawTimeExpr + "|" + ev.Title))
		if !seen[key] {
			all = append(all, ev)
		}
	}
	return all
}

func (a *Agent) extractByLLM(ctx context.Context, content, chapterID string) []extractedEvent {
	if a.llm == nil {
		return nil
	}

	// Truncate content to avoid token limits
	runes := []rune(content)
	if len(runes) > 3000 {
		runes = runes[:3000]
	}
	truncated := string(runes)

	prompt := fmt.Sprintf(`请从以下章节文本中提取所有与时间相关的事件。对每个事件，返回JSON数组，每个元素包含：
- "title": 事件标题（简短概括）
- "summary": 事件摘要（从原文截取关键描述，50字以内）
- "event_name": 事件名称标识（用于合并同一事件在不同章节的描述，如"青云宗大比"）
- "time_expr": 原始时间表述（如"第二天"、"三天后"）
- "offset_hint": 时间偏移提示（数字+单位，如"3天"、"1个月"，无法判断则为"unknown"）

只返回JSON数组，不要其他说明。如果没有时间相关事件，返回空数组 []。

章节内容：
%s`, truncated)

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是小说时间线分析助手。从文本中抽取时间事件，返回JSON格式。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.2, MaxTokens: 2048})
	if err != nil {
		return nil
	}

	// Parse response
	resp = strings.TrimSpace(resp)
	// Strip markdown code fences if present
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		var clean []string
		for _, l := range lines {
			if strings.HasPrefix(l, "```") {
				continue
			}
			clean = append(clean, l)
		}
		resp = strings.Join(clean, "\n")
		resp = strings.TrimSpace(resp)
	}

	var raw []struct {
		Title      string `json:"title"`
		Summary    string `json:"summary"`
		EventName  string `json:"event_name"`
		TimeExpr   string `json:"time_expr"`
		OffsetHint string `json:"offset_hint"`
	}
	if err := json.Unmarshal([]byte(resp), &raw); err != nil {
		return nil
	}

	var events []extractedEvent
	for _, r := range raw {
		if r.Title == "" {
			continue
		}
		offset := parseOffsetHint(r.OffsetHint)
		events = append(events, extractedEvent{
			Offset:      offset,
			Label:       r.Title,
			Title:       r.Title,
			Summary:     r.Summary,
			ChapterIDs:  []string{chapterID},
			EventName:   r.EventName,
			RawTimeExpr: r.TimeExpr,
		})
	}
	return events
}

// parseOffsetHint converts "3天" / "1个月" / "unknown" to days.
func parseOffsetHint(hint string) float64 {
	hint = strings.TrimSpace(hint)
	if hint == "" || hint == "unknown" {
		return -1
	}
	var val float64
	var unit string
	fmt.Sscanf(hint, "%f%s", &val, &unit)
	switch {
	case strings.Contains(unit, "年"):
		return val * 365
	case strings.Contains(unit, "月"):
		return val * 30
	case strings.Contains(unit, "时辰"):
		return val / 12
	case strings.Contains(unit, "天") || strings.Contains(unit, "日"):
		return val
	default:
		return val
	}
}
