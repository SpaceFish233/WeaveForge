package style

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"weaveforge/internal/llm"
	"weaveforge/internal/textutil"
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

// DetectAIFlavor runs a five-dimension Anti-AI flavor check on the given text.
func (a *Agent) DetectAIFlavor(ctx context.Context, text string) (AIFlavorReport, error) {
	if strings.TrimSpace(text) == "" {
		return AIFlavorReport{}, fmt.Errorf("style: empty text for AI flavor detection")
	}
	prompt := buildAIFlavorPrompt(text)
	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: aiFlavorSystemPrompt},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.1, MaxTokens: 4096})
	if err != nil {
		return AIFlavorReport{}, fmt.Errorf("style: ai flavor llm: %w", err)
	}

	var report AIFlavorReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &report); err != nil {
		// Try to extract JSON from response if LLM added extra text
		start := strings.Index(resp, "{")
		end := strings.LastIndex(resp, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(resp[start:end+1]), &report); err2 != nil {
				return AIFlavorReport{}, fmt.Errorf("style: parse ai flavor result: %w (raw: %.200s)", err, resp)
			}
		} else {
			return AIFlavorReport{}, fmt.Errorf("style: parse ai flavor result: %w (raw: %.200s)", err, resp)
		}
	}
	report.CheckedAt = time.Now().Format("2006-01-02 15:04:05")
	return report, nil
}

// CheckChapterHook analyzes the last ~500 characters of chapter content for hook quality.
func (a *Agent) CheckChapterHook(ctx context.Context, chapterContent, prevChapterContent string) (HookCheckResult, error) {
	if strings.TrimSpace(chapterContent) == "" {
		return HookCheckResult{}, fmt.Errorf("style: empty chapter content for hook check")
	}
	prompt := buildHookCheckPrompt(chapterContent, prevChapterContent)
	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: hookCheckSystemPrompt},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.2, MaxTokens: 2048})
	if err != nil {
		return HookCheckResult{}, fmt.Errorf("style: hook check llm: %w", err)
	}

	var result HookCheckResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &result); err != nil {
		start := strings.Index(resp, "{")
		end := strings.LastIndex(resp, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(resp[start:end+1]), &result); err2 != nil {
				return HookCheckResult{}, fmt.Errorf("style: parse hook result: %w (raw: %.200s)", err, resp)
			}
		} else {
			return HookCheckResult{}, fmt.Errorf("style: parse hook result: %w (raw: %.200s)", err, resp)
		}
	}
	result.CheckedAt = time.Now().Format("2006-01-02 15:04:05")
	return result, nil
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

// ─── Anti-AI Flavor Detection ─────────────────────────────────────────

const aiFlavorSystemPrompt = `你是一位专业的网络小说文本分析专家。你的任务是检测文本中的"AI写作痕迹"，从五个维度逐一检查，输出严格的JSON格式报告。

核心原则：你只找问题、给证据、给修复方向。不评价文笔好坏，不给出主观感受。每个问题必须有原文引用作为证据。`

func buildAIFlavorPrompt(text string) string {
	runes := []rune(text)
	var excerpt string
	if len(runes) > 3000 {
		excerpt = string(runes[:3000]) + "\n\n... (文本过长，仅分析前3000字)"
	} else {
		excerpt = text
	}
	return fmt.Sprintf(`请从以下五个维度逐一检查这段网络小说文本的"AI写作痕迹"，输出一份严格的JSON报告。

【待检测文本】
%s

【五维检查清单】

1. 词汇层（category: vocabulary）
- 高频AI词汇是否密集（如"缓缓/淡淡/微微/轻轻"+"动词"结构在500字内出现3次以上）
- 是否大量使用神态模板（"眸中闪过一丝X""瞳孔微缩""心中一凛""嘴角扬起一抹X"）
- 是否滥用万能副词修饰动作
- severity: 个别命中=low，密集命中=high

2. 句式层（category: sentence）
- 是否存在"起因→经过→结果→感悟"四段闭环结构
- 是否存在连续3句以上主谓宾结构一致的同构句
- 是否每段都以总结句收尾（"他终于明白了...""由此可见...""更重要的是..."）
- 是否存在同一信息用不同句式重复说2-3遍
- severity: high

3. 叙事层（category: narrative）
- 段落信息密度是否过于均匀，无快慢之分
- 是否存在"他不知道的是...""殊不知..."等戏剧性反讽提示
- 章末是否"安全着陆"（所有冲突完美解决，无遗留不安感或未闭合问题）
- 是否存在展示后紧跟解释（用动作展示后紧接着一句话解释刚才动作的含义）
- severity: medium

4. 情感层（category: emotion）
- 情绪描写是否标签化（"他感到愤怒""她非常紧张"而非通过行为和生理反应展示）
- 是否存在情绪即时切换（上句愤怒下句平静，无过渡）
- 所有角色是否用同一套反应模板
- severity: 标签化=high，其他=medium

5. 对话层（category: dialogue）
- 对话是否为信息宣讲（解释背景而非推进冲突）
- 是否全员书面语、无口语特征、无个人口癖
- 对白后是否紧跟解释性叙述（"他这么说是因为..."）
- severity: 信息宣讲=high，其他=medium

【输出格式】
严格输出以下JSON，不要任何额外文本：
{
  "dimensions": [
    {
      "label": "词汇层",
      "severity": "pass|low|medium|high",
      "issues": [
        {"description": "具体问题描述", "evidence": "原文引用", "fix_hint": "修复方向"}
      ]
    },
    {"label": "句式层", "severity": "...", "issues": [...]},
    {"label": "叙事层", "severity": "...", "issues": [...]},
    {"label": "情感层", "severity": "...", "issues": [...]},
    {"label": "对话层", "severity": "...", "issues": [...]}
  ],
  "summary": "一句话总结：共发现X个维度有AI痕迹，其中最突出的问题是..."
}

注意：
- 如果某维度没有问题，severity为"pass"，issues为空数组[]
- 每个issue的evidence必须是原文中的具体句子或短语引用，不能是"多处""很多"等模糊描述
- fix_hint要给出可操作的修改建议，而不是"需要改进"等空话`, excerpt)
}

// ─── Chapter Hook Check ───────────────────────────────────────────────

const hookCheckSystemPrompt = `你是一位专业的网络小说节奏分析专家。你只分析章节结尾的钩子质量，给出客观的结构化评估。不评价文笔，不给出主观感受。`

func buildHookCheckPrompt(chapterContent, prevChapterContent string) string {
	// Take last ~800 runes of chapter for ending analysis
	runes := []rune(chapterContent)
	var ending string
	if len(runes) > 800 {
		ending = "...(前文省略)\n\n" + string(runes[len(runes)-800:])
	} else {
		ending = chapterContent
	}

	var prevEnding string
	if prevChapterContent != "" {
		prevRunes := []rune(prevChapterContent)
		if len(prevRunes) > 200 {
			prevEnding = string(prevRunes[len(prevRunes)-200:])
		} else {
			prevEnding = prevChapterContent
		}
	}

	var prevSection string
	if prevEnding != "" {
		prevSection = fmt.Sprintf("【上一章结尾】\n%s\n\n", prevEnding)
	}

	return fmt.Sprintf(`请分析这章结尾的钩子质量，判断读者翻到下一章的欲望有多强。

%s【本章结尾内容】
%s

【钩子类型定义】
- 危机钩（crisis）：危险逼近、敌人出现、时间紧迫
- 悬念钩（mystery）：信息缺口、未解之谜、反常现象
- 渴望钩（desire）：好事即将发生、奖励可期、升级在望
- 情绪钩（emotion）：愤怒、心疼、共情、心动等强烈情绪
- 选择钩（choice）：两难抉择、高风险决策
- 认知钩（recognition）：身份揭晓、真相浮现、认知颠覆
- 无钩子（none）：章末所有冲突已解决，没有留给读者的悬念

【分析要点】
1. 本章最后几段是否留下了一个未解决的问题或期待？
2. 如果上章有钩子承诺，本章结尾是否回应了？
3. 钩子强度：weak（有悬念但不够抓人）、medium（有明确的阅读驱动）、strong（读者必须翻下一章）

【输出格式】
严格输出以下JSON：
{
  "has_hook": true,
  "hook_type": "crisis|mystery|desire|emotion|choice|recognition|none",
  "hook_strength": "weak|medium|strong",
  "unresolved_questions": ["本章结尾留下的具体未解问题1", "问题2"],
  "closing_analysis": "对本章结尾段落的分析，指出钩子是如何埋设的或为什么没有钩子",
  "suggestion": "如何加强章末钩子的具体建议，如果没有钩子则说明可以从哪个方向设置"
}`, prevSection, ending)
}

// ─── Pre-Write Constraint Check ──────────────────────────────────────

const constraintCheckSystemPrompt = `你是一位专业的小说设定审核员。你的任务是逐条核对章节正文是否违反了已建立的设定约束，输出严格的结构化JSON报告。三大定律：大纲即法律、设定即物理、发明需识别。你只找矛盾、给证据、给修复方向。`

// CheckWritingConstraints verifies chapter content against established settings,
// characters, and outline constraints. It loads all constraint sources from DB.
func (a *Agent) CheckWritingConstraints(ctx context.Context, chapterContent string) (ConstraintCheckResult, error) {
	if strings.TrimSpace(chapterContent) == "" {
		return ConstraintCheckResult{}, fmt.Errorf("style: empty chapter content for constraint check")
	}

	constraintCtx := a.buildConstraintContext(ctx)
	prompt := buildConstraintCheckPrompt(chapterContent, constraintCtx)
	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: constraintCheckSystemPrompt},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.1, MaxTokens: 4096})
	if err != nil {
		return ConstraintCheckResult{}, fmt.Errorf("style: constraint check llm: %w", err)
	}

	var result ConstraintCheckResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &result); err != nil {
		start := strings.Index(resp, "{")
		end := strings.LastIndex(resp, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(resp[start:end+1]), &result); err2 != nil {
				return ConstraintCheckResult{}, fmt.Errorf("style: parse constraint result: %w", err)
			}
		} else {
			return ConstraintCheckResult{}, fmt.Errorf("style: parse constraint result: %w", err)
		}
	}
	result.CheckedAt = time.Now().Format("2006-01-02 15:04:05")
	// Count blocking issues
	for _, iss := range result.Issues {
		if iss.Severity == "critical" {
			result.BlockingCount++
		}
	}
	result.Passed = result.BlockingCount == 0
	return result, nil
}

func (a *Agent) buildConstraintContext(ctx context.Context) string {
	var sb strings.Builder

	// Load settings (world rules, power systems, etc.)
	var settings []models.WorldSetting
	if err := a.db.WithContext(ctx).Order("type asc, title asc").Find(&settings).Error; err == nil && len(settings) > 0 {
		sb.WriteString("=== 已建立的世界设定 ===\n")
		for _, s := range settings {
			content := strings.TrimSpace(s.Content)
			if len([]rune(content)) > 300 {
				content = string([]rune(content)[:300]) + "…"
			}
			fmt.Fprintf(&sb, "【%s - %s】\n%s\n\n", s.Type, s.Title, content)
		}
	}

	// Load active characters
	var characters []models.Character
	if err := a.db.WithContext(ctx).Where("status = ?", "active").Order("role asc, name asc").Find(&characters).Error; err == nil && len(characters) > 0 {
		sb.WriteString("=== 当前角色状态 ===\n")
		for _, c := range characters {
			fmt.Fprintf(&sb, "【%s】%s (%s) — %s — %s\n", c.Role, c.Name, c.Gender, c.Personality, c.Description)
		}
		sb.WriteString("\n")
	}

	// Load outline tree
	var nodes []models.OutlineNode
	if err := a.db.WithContext(ctx).Order("sort_order asc").Find(&nodes).Error; err == nil && len(nodes) > 0 {
		sb.WriteString("=== 大纲结构 ===\n")
		for _, n := range nodes {
			if n.Title != "" {
				statusMark := ""
				switch n.Status {
				case "completed":
					statusMark = " [已完成]"
				case "in_progress":
					statusMark = " [进行中]"
				}
				if n.Summary != "" {
					fmt.Fprintf(&sb, "• %s%s: %s\n", n.Title, statusMark, n.Summary)
				} else {
					fmt.Fprintf(&sb, "• %s%s\n", n.Title, statusMark)
				}
			}
		}
	}

	if sb.Len() == 0 {
		return "（项目尚未建立设定集、角色库或大纲，仅检查文本内部自洽性）"
	}
	return sb.String()
}

func buildConstraintCheckPrompt(chapterContent string, constraintCtx string) string {
	runes := []rune(chapterContent)
	var excerpt string
	if len(runes) > 3000 {
		excerpt = string(runes[:3000]) + "\n\n... (文本过长，仅分析前3000字)"
	} else {
		excerpt = chapterContent
	}
	return fmt.Sprintf(`请对照以下约束上下文，检查本章正文是否存在违反设定的问题。

【约束上下文——必须遵守的规则】
%s

【待检查章节正文】
%s

【检查维度】
1. 设定一致性（setting）：角色能力是否与当前境界匹配？地点描述是否与世界观一致？物品/货币使用是否符合已建立规则？
2. 角色一致性（character）：对话风格是否符合角色特征？行为是否与已建立的性格/动机一致？角色是否使用了不应知道的信息？
3. 时间线（timeline）：本章时间是否与已知时间体系一致？事件顺序是否合理？
4. 逻辑（logic）：因果关系是否成立？角色决策是否有合理动机？冲突结果是否符合力量对比？
5. 连续性（continuity）：本章情节是否与大纲规划一致？是否出现与已完成章节明显矛盾的内容？

【严重性定义】
- critical：确定的事实矛盾，必须修复（如角色能力超出已建立上限、已死角色复活）
- high：极可能存在矛盾，强烈建议修复
- medium：可能存在不一致，建议人工确认
- low：轻微偏差，可忽略

【输出格式】
严格输出以下JSON：
{
  "passed": true,
  "issues": [
    {
      "category": "setting",
      "severity": "critical|high|medium|low",
      "description": "具体问题描述",
      "evidence": "原文引用 vs 约束要求",
      "fix_hint": "修复方向"
    }
  ],
  "summary": "一句话总结检查结果"
}

注意：
- 如果没有任何问题，issues为空数组[]，passed为true
- 不要报告"风格可以优化""这里可以写得更好"等非约束性问题
- 只有当正文与约束上下文存在可验证的矛盾时才报告issue`, constraintCtx, excerpt)
}

// ─── Placeholder Scan ────────────────────────────────────────────────

var placeholderPatterns = []struct {
	re      *regexp.Regexp
	pType   string
}{
	{regexp.MustCompile(`\[(TODO|待补充|待完善|待定|待填|占位|略|补充|填写)[^\]]*\]`), "todo"},
	{regexp.MustCompile(`\{[^}}]*(占位|待定|待填|补充|TODO|todo)[^}]*\}`), "placeholder"},
	{regexp.MustCompile(`（(待补充|待定|占位)）`), "placeholder"},
	{regexp.MustCompile(`\b(暂名|未命名|佚名|无名)[的]?`), "temp_name"},
	{regexp.MustCompile(`……+\s*(待续|未完|省略)`), "ellipsis"},
	{regexp.MustCompile(`\S*？{2,}\S*`), "placeholder"}, // double-width ??  pattern
	{regexp.MustCompile(`\[[Nn]ame\]|\[[Dd]ate\]|\[[Pp]lace\]|\[[Nn]umber\]`), "todo"},
}

// ScanPlaceholders scans text for unfinished markers without using LLM.
func (a *Agent) ScanPlaceholders(text string) PlaceholderScanResult {
	result := PlaceholderScanResult{Clean: true, Matches: []PlaceholderMatch{}, CheckedAt: time.Now().Format("2006-01-02 15:04:05")}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return result
	}

	runes := []rune(trimmed)
	for _, pp := range placeholderPatterns {
		matches := pp.re.FindAllStringIndex(trimmed, -1)
		for _, m := range matches {
			// Convert byte offsets to rune offsets
			byteToRune := textutil.ByteToRuneMap(trimmed)
			rStart := byteToRune[m[0]]
			rEnd := byteToRune[m[1]]

			ctxStart := rStart - 15
			if ctxStart < 0 {
				ctxStart = 0
			}
			ctxEnd := rEnd + 25
			if ctxEnd > len(runes) {
				ctxEnd = len(runes)
			}

			result.Matches = append(result.Matches, PlaceholderMatch{
				Pattern:    string(runes[rStart:rEnd]),
				Type:       pp.pType,
				StartIndex: rStart,
				EndIndex:   rEnd,
				Context:    strings.TrimSpace(string(runes[ctxStart:ctxEnd])),
			})
		}
	}

	if len(result.Matches) > 0 {
		result.Clean = false
	}
	return result
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
