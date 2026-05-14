package style

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

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

// ─── Public API ────────────────────────────────────────────────────────

func (a *Agent) LearnStyle(ctx context.Context, name string, chapterIDs []string) (string, error) {
	if len(chapterIDs) == 0 {
		return "", fmt.Errorf("style: no chapters provided")
	}
	var chapters []models.Chapter
	if err := a.db.WithContext(ctx).Where("id IN ?", chapterIDs).Order("sort_order asc").Find(&chapters).Error; err != nil {
		return "", fmt.Errorf("style: load chapters: %w", err)
	}
	if len(chapters) == 0 {
		return "", fmt.Errorf("style: no valid chapters found")
	}
	var fullText strings.Builder
	for _, ch := range chapters {
		fullText.WriteString(ch.Content)
		fullText.WriteString("\n")
	}
	features := analyzeStyle(fullText.String())
	samples := extractSamples(fullText.String(), 3)
	desc := buildStyleDescription(features)

	sourceIDs, _ := json.Marshal(chapterIDs)
	featuresJSON, _ := json.Marshal(features)
	samplesJSON, _ := json.Marshal(samples)

	profile := models.StyleProfile{
		ID:             uuid.New().String(),
		Name:           name,
		Features:       string(featuresJSON),
		Samples:        string(samplesJSON),
		Description:    desc,
		SourceChapters: string(sourceIDs),
	}
	if err := a.db.WithContext(ctx).Create(&profile).Error; err != nil {
		return "", fmt.Errorf("style: save profile: %w", err)
	}
	return profile.ID, nil
}

func (a *Agent) ListProfiles(ctx context.Context) ([]StyleProfileSummary, error) {
	var rows []models.StyleProfile
	if err := a.db.WithContext(ctx).Order("created_at desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	summaries := make([]StyleProfileSummary, len(rows))
	for i, r := range rows {
		desc := r.Description
		if len([]rune(desc)) > 80 {
			desc = string([]rune(desc)[:80]) + "..."
		}
		summaries[i] = StyleProfileSummary{
			ID:          r.ID,
			Name:        r.Name,
			Description: desc,
			CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04"),
		}
	}
	return summaries, nil
}

func (a *Agent) GetProfile(ctx context.Context, id string) (*models.StyleProfile, error) {
	var p models.StyleProfile
	if err := a.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (a *Agent) DeleteProfile(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Delete(&models.StyleProfile{}, "id = ?", id).Error
}

func (a *Agent) PolishText(ctx context.Context, text, profileID, intensity string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("style: empty text")
	}
	var profile models.StyleProfile
	if err := a.db.WithContext(ctx).First(&profile, "id = ?", profileID).Error; err != nil {
		return "", fmt.Errorf("style: profile not found: %w", err)
	}
	var features StyleFeatures
	if err := json.Unmarshal([]byte(profile.Features), &features); err != nil {
		return "", fmt.Errorf("style: parse features: %w", err)
	}
	var samples []string
	json.Unmarshal([]byte(profile.Samples), &samples)

	prompt := buildPolishPrompt(profile.Name, profile.Description, features, samples, intensity, text)
	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是一位专业的网络小说润色助手。按要求对文本进行润色，只输出润色后的结果，不要添加任何说明。"},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.3, MaxTokens: 4096})
	if err != nil {
		return "", fmt.Errorf("style: polish llm: %w", err)
	}
	return strings.TrimSpace(resp), nil
}

func (a *Agent) PolishWithInstruction(ctx context.Context, text, instruction, profileID string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("style: empty text")
	}

	var sb strings.Builder
	sb.WriteString("请根据以下要求对文本进行润色。\n\n")

	// If a style profile is provided, include its context
	if profileID != "" {
		var profile models.StyleProfile
		if err := a.db.WithContext(ctx).First(&profile, "id = ?", profileID).Error; err == nil {
			sb.WriteString("【参考风格】\n")
			sb.WriteString(profile.Description)
			sb.WriteString("\n\n")
			var samples []string
			json.Unmarshal([]byte(profile.Samples), &samples)
			if len(samples) > 0 {
				sb.WriteString("【风格样本】\n")
				for i, s := range samples {
					if i >= 2 {
						break
					}
					trimmed := strings.TrimSpace(s)
					if len([]rune(trimmed)) > 200 {
						trimmed = string([]rune(trimmed)[:200]) + "…"
					}
					sb.WriteString(trimmed)
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString("【润色要求】\n")
	sb.WriteString(instruction)
	sb.WriteString("\n\n【原文】\n")
	sb.WriteString(text)
	sb.WriteString("\n\n请直接输出润色后的文本，不要添加任何说明、前缀或后缀。")

	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是一位专业的网络小说润色助手。按要求对文本进行润色，只输出润色后的结果，不要添加任何说明。"},
		{Role: "user", Content: sb.String()},
	}, a.chatModel, llm.ChatOption{Temperature: 0.3, MaxTokens: 4096})
	if err != nil {
		return "", fmt.Errorf("style: polish llm: %w", err)
	}
	return strings.TrimSpace(resp), nil
}

func (a *Agent) AnalyzeStyle(ctx context.Context, text, profileID string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("style: empty text")
	}
	var profile models.StyleProfile
	if err := a.db.WithContext(ctx).First(&profile, "id = ?", profileID).Error; err != nil {
		return "", fmt.Errorf("style: profile not found: %w", err)
	}
	var sb strings.Builder
	sb.WriteString("【目标风格】\n")
	sb.WriteString(profile.Description)
	sb.WriteString("\n\n【待分析文本】\n")
	sb.WriteString(text)
	sb.WriteString("\n\n请分析这段文本与目标风格的相符程度和差异。从以下维度分析：\n")
	sb.WriteString("1. 句子长度和节奏\n2. 用词风格和词汇丰富度\n")
	sb.WriteString("3. 对话与叙述比例\n4. 修辞手法使用\n")
	sb.WriteString("5. 段落组织方式\n6. 整体风格匹配度\n")
	sb.WriteString("给出具体分析和改进建议。")
	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: "你是一位文学风格分析专家。根据目标风格档案分析给定文本，给出细致客观的风格差异报告。"},
		{Role: "user", Content: sb.String()},
	}, a.chatModel, llm.ChatOption{Temperature: 0.2})
	if err != nil {
		return "", fmt.Errorf("style: analyze llm: %w", err)
	}
	return strings.TrimSpace(resp), nil
}

// ─── Style Analysis ────────────────────────────────────────────────────

var rhetoricWords = []string{"像", "仿佛", "如同", "似乎", "宛如", "好像", "一般", "似的", "般", "犹如", "好比", "类似"}

func analyzeStyle(text string) StyleFeatures {
	runes := []rune(text)
	totalChars := len(runes)
	if totalChars == 0 {
		return StyleFeatures{}
	}

	sentences := splitSentences(text)
	totalSentences := len(sentences)
	if totalSentences == 0 {
		totalSentences = 1
	}
	paragraphs := splitParagraphs(text)
	totalParagraphs := len(paragraphs)
	if totalParagraphs == 0 {
		totalParagraphs = 1
	}

	var sentLengths []float64
	var sentLengthSum float64
	for _, s := range sentences {
		l := float64(utf8.RuneCountInString(strings.TrimSpace(s)))
		sentLengths = append(sentLengths, l)
		sentLengthSum += l
	}
	avgSentLen := sentLengthSum / float64(totalSentences)
	var variance float64
	for _, l := range sentLengths {
		d := l - avgSentLen
		variance += d * d
	}
	sentStd := math.Sqrt(variance / float64(totalSentences))

	avgParaLen := float64(totalChars) / float64(totalParagraphs)

	// Dialogue ratio: count chars inside corner brackets
	dialogueChars := 0
	inDialogue := false
	for _, r := range runes {
		if r == 0x300C { // 「
			inDialogue = true
			dialogueChars++
		} else if r == 0x300D { // 」
			inDialogue = false
			dialogueChars++
		} else if inDialogue {
			dialogueChars++
		}
	}
	dialogueRatio := float64(dialogueChars) / float64(totalChars)

	// Vocabulary
	charSet := make(map[rune]int)
	for _, r := range runes {
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) {
			charSet[r]++
		}
	}
	totalContentChars := 0
	for _, count := range charSet {
		totalContentChars += count
	}
	richness := float64(len(charSet)) / float64(totalContentChars)

	// Top chars
	type cc struct {
		c rune
		n int
	}
	var ccList []cc
	for r, n := range charSet {
		ccList = append(ccList, cc{r, n})
	}
	sort.Slice(ccList, func(i, j int) bool { return ccList[i].n > ccList[j].n })
	topN := 20
	if len(ccList) < topN {
		topN = len(ccList)
	}
	topChars := make([]CharFreq, topN)
	for i := 0; i < topN; i++ {
		topChars[i] = CharFreq{Char: string(ccList[i].c), Freq: float64(ccList[i].n) / float64(totalContentChars)}
	}

	// Bigram frequency
	bigramSet := make(map[string]int)
	for i := 0; i < len(runes)-1; i++ {
		if !unicode.IsPunct(runes[i]) && !unicode.IsSpace(runes[i]) &&
			!unicode.IsPunct(runes[i+1]) && !unicode.IsSpace(runes[i+1]) {
			bigramSet[string(runes[i:i+2])]++
		}
	}
	type bc struct {
		s string
		n int
	}
	var blist []bc
	for s, n := range bigramSet {
		blist = append(blist, bc{s, n})
	}
	sort.Slice(blist, func(i, j int) bool { return blist[i].n > blist[j].n })
	topBN := 20
	if len(blist) < topBN {
		topBN = len(blist)
	}
	topBigrams := make([]CharFreq, topBN)
	totalBigramOccurrences := 0
	for _, b := range blist {
		totalBigramOccurrences += b.n
	}
	if totalBigramOccurrences == 0 {
		totalBigramOccurrences = 1
	}
	for i := 0; i < topBN; i++ {
		topBigrams[i] = CharFreq{Char: blist[i].s, Freq: float64(blist[i].n) / float64(totalBigramOccurrences)}
	}

	// Punctuation
	punctFreq := make(map[string]float64)
	totalPunct := 0
	for _, r := range runes {
		if unicode.IsPunct(r) {
			punctFreq[string(r)]++
			totalPunct++
		}
	}
	if totalPunct > 0 {
		for k, v := range punctFreq {
			punctFreq[k] = v / float64(totalPunct)
		}
	}

	// Rhetoric markers
	rhetoricCount := 0
	for _, w := range rhetoricWords {
		rhetoricCount += strings.Count(text, w)
	}
	rhetoricDensity := float64(rhetoricCount) / float64(totalChars) * 1000

	return StyleFeatures{
		AvgSentenceLength:   math.Round(avgSentLen*100) / 100,
		AvgParagraphLength:  math.Round(avgParaLen*100) / 100,
		SentenceLengthStd:   math.Round(sentStd*100) / 100,
		DialogueRatio:       math.Round(dialogueRatio*10000) / 10000,
		VocabularyRichness:  math.Round(richness*10000) / 10000,
		TotalChars:          totalChars,
		TotalSentences:      totalSentences,
		TotalParagraphs:     totalParagraphs,
		TopChars:            topChars,
		TopBigrams:          topBigrams,
		PunctuationFreq:     punctFreq,
		RhetoricDensity:     math.Round(rhetoricDensity*100) / 100,
	}
}

func buildStyleDescription(f StyleFeatures) string {
	var sb strings.Builder
	sb.WriteString("写作风格分析报告：\n")

	if f.AvgSentenceLength < 20 {
		sb.WriteString(fmt.Sprintf("• 句式：句子偏短（平均 %.1f 字），节奏明快\n", f.AvgSentenceLength))
	} else if f.AvgSentenceLength < 40 {
		sb.WriteString(fmt.Sprintf("• 句式：句子长度中等（平均 %.1f 字），节奏适中\n", f.AvgSentenceLength))
	} else {
		sb.WriteString(fmt.Sprintf("• 句式：句子偏长（平均 %.1f 字），节奏舒缓\n", f.AvgSentenceLength))
	}
	if f.SentenceLengthStd < 10 {
		sb.WriteString("• 句式变化：句子长度较均匀\n")
	} else if f.SentenceLengthStd < 25 {
		sb.WriteString("• 句式变化：句子长度有一定变化\n")
	} else {
		sb.WriteString("• 句式变化：句子长度变化丰富\n")
	}
	if f.DialogueRatio < 0.2 {
		sb.WriteString(fmt.Sprintf("• 对话占比：%.0f%%，偏重叙述描写\n", f.DialogueRatio*100))
	} else if f.DialogueRatio < 0.5 {
		sb.WriteString(fmt.Sprintf("• 对话占比：%.0f%%，叙述与对话均衡\n", f.DialogueRatio*100))
	} else {
		sb.WriteString(fmt.Sprintf("• 对话占比：%.0f%%，偏重对话驱动\n", f.DialogueRatio*100))
	}
	if f.VocabularyRichness < 0.1 {
		sb.WriteString("• 用词：词汇丰富度较高\n")
	} else if f.VocabularyRichness < 0.2 {
		sb.WriteString("• 用词：词汇丰富度中等\n")
	} else {
		sb.WriteString("• 用词：词汇较为集中，风格统一\n")
	}
	if f.RhetoricDensity < 1 {
		sb.WriteString("• 修辞：修辞使用较少，风格平实\n")
	} else if f.RhetoricDensity < 3 {
		sb.WriteString("• 修辞：适度使用修辞手法\n")
	} else {
		sb.WriteString("• 修辞：使用较多修辞手法，文采丰富\n")
	}
	if len(f.TopChars) >= 3 {
		sb.WriteString(fmt.Sprintf("• 高频字词：「%s」「%s」「%s」等\n",
			f.TopChars[0].Char, f.TopChars[1].Char, f.TopChars[2].Char))
	}
	return sb.String()
}

func buildPolishPrompt(name, desc string, features StyleFeatures, samples []string, intensity, text string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("请按照以下风格档案对文本进行润色。\n\n【风格名称】%s\n\n", name))
	sb.WriteString(desc)
	sb.WriteString("\n")

	if len(features.TopBigrams) > 0 {
		sb.WriteString("常用搭配词：")
		n := 10
		if len(features.TopBigrams) < n {
			n = len(features.TopBigrams)
		}
		for i := 0; i < n; i++ {
			sb.WriteString(features.TopBigrams[i].Char)
			if i < n-1 {
				sb.WriteString("、")
			}
		}
		sb.WriteString("\n\n")
	}

	if len(samples) > 0 {
		sb.WriteString("【风格参考样本】\n")
		for i, s := range samples {
			if i >= 2 {
				break
			}
			trimmed := strings.TrimSpace(s)
			if len([]rune(trimmed)) > 200 {
				trimmed = string([]rune(trimmed)[:200]) + "…"
			}
			sb.WriteString(fmt.Sprintf("样本 %d：\n%s\n\n", i+1, trimmed))
		}
	}

	switch intensity {
	case "轻微":
		sb.WriteString("【润色强度】轻微——仅修正语病、错别字和明显不通顺之处，尽可能保持原文措辞。\n")
	case "中等":
		sb.WriteString("【润色强度】中等——在保持原意的前提下优化表达，使风格更贴近目标，但不做大范围改写。\n")
	case "较大":
		sb.WriteString("【润色强度】较大——在保持核心内容和人物设定的前提下，大幅改写以匹配目标风格。\n")
	default:
		sb.WriteString("【润色强度】中等——在保持原意的同时优化表达以贴近目标风格。\n")
	}
	sb.WriteString("\n【待润色文本】\n")
	sb.WriteString(text)
	sb.WriteString("\n\n请直接输出润色后的文本，不要添加任何说明、前缀或后缀。")
	return sb.String()
}

// ─── Text Utilities ────────────────────────────────────────────────────

func splitSentences(text string) []string {
	var sentences []string
	var buf strings.Builder
	for _, r := range text {
		buf.WriteRune(r)
		if r == 0x3002 || r == 0xFF01 || r == 0xFF1F || r == '\n' { // 。！？
			s := strings.TrimSpace(buf.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			sentences = append(sentences, s)
		}
	}
	return sentences
}

func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n")
	var paras []string
	for _, p := range raw {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			paras = append(paras, trimmed)
		}
	}
	return paras
}

func extractSamples(text string, n int) []string {
	paras := splitParagraphs(text)
	var candidates []string
	for _, p := range paras {
		if utf8.RuneCountInString(p) >= 40 {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) > n {
		step := float64(len(candidates)) / float64(n)
		var selected []string
		for i := 0; i < n; i++ {
			idx := int(float64(i) * step)
			if idx >= len(candidates) {
				idx = len(candidates) - 1
			}
			selected = append(selected, candidates[idx])
		}
		return selected
	}
	return candidates
}
