package typo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"weaveforge/internal/llm"
)

const (
	maxSegmentRunes = 1500
	minContentRunes = 5
)

type Agent struct {
	llm       chatClient
	chatModel string
}

func NewAgent(llm chatClient, chatModel string) *Agent {
	return &Agent{llm: llm, chatModel: chatModel}
}

// DetectTypos checks the given content for typos using LLM.
// It segments long content by paragraphs and processes each segment independently.
// Returns TypoSuggestion with sentence-relative indices (StartIndex/EndIndex are
// relative to the Sentence field, not to the full content).
func (a *Agent) DetectTypos(ctx context.Context, content string) ([]TypoSuggestion, error) {
	if a.llm == nil || !a.llm.HasConfig() {
		return nil, fmt.Errorf("typo: 未配置 LLM API，请先在设置中配置 API Key 和 Base URL")
	}
	if strings.TrimSpace(a.chatModel) == "" {
		return nil, fmt.Errorf("typo: 未配置 LLM 模型，请先在设置中选择模型")
	}

	runes := []rune(strings.TrimSpace(content))
	log.Printf("[Typo] DetectTypos called, content length: %d runes, chatModel: %q", len(runes), a.chatModel)
	if len(runes) < minContentRunes {
		log.Printf("[Typo] Content too short (%d < %d), returning empty", len(runes), minContentRunes)
		return nil, nil
	}

	segments := splitByParagraph(runes, maxSegmentRunes)
	log.Printf("[Typo] Split into %d segments", len(segments))
	var allResults []TypoSuggestion

	for i, seg := range segments {
		log.Printf("[Typo] Processing segment %d/%d (%d runes)", i+1, len(segments), len(seg))
		suggestions, err := a.detectSegment(ctx, string(seg))
		if err != nil {
			log.Printf("[Typo] Segment %d failed: %v", i+1, err)
			return nil, fmt.Errorf("typo: 检测失败 (段 %d/%d): %w", i+1, len(segments), err)
		}
		log.Printf("[Typo] Segment %d returned %d suggestions", i+1, len(suggestions))
		allResults = append(allResults, suggestions...)
	}

	log.Printf("[Typo] Total results: %d", len(allResults))
	return allResults, nil
}

// detectSegment sends a single segment to the LLM for typo detection.
func (a *Agent) detectSegment(ctx context.Context, segment string) ([]TypoSuggestion, error) {
	prompt := buildDetectPrompt(segment)
	resp, err := a.llm.ChatCompletion(ctx, []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}, a.chatModel, llm.ChatOption{Temperature: 0.1, MaxTokens: 4096})
	if err != nil {
		return nil, fmt.Errorf("typo: LLM 调用失败: %w", err)
	}
	if strings.TrimSpace(resp) == "" {
		return nil, fmt.Errorf("typo: LLM 返回为空，请检查 API 配置是否正确")
	}

	results, err := parseResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("typo: 解析 LLM 响应失败: %w", err)
	}
	return results, nil
}

// splitByParagraph splits runes into segments by paragraph boundaries (\n),
// ensuring no segment exceeds maxRunes. Long paragraphs are split by sentences.
func splitByParagraph(runes []rune, maxRunes int) [][]rune {
	var segments [][]rune
	lines := splitLines(runes)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if len(line) <= maxRunes {
			segments = append(segments, line)
		} else {
			// Split long paragraph by sentence boundaries
			subSegments := splitBySentences(line, maxRunes)
			segments = append(segments, subSegments...)
		}
	}
	return segments
}

func splitLines(runes []rune) [][]rune {
	var lines [][]rune
	start := 0
	for i, r := range runes {
		if r == '\n' {
			lines = append(lines, runes[start:i])
			start = i + 1
		}
	}
	if start < len(runes) {
		lines = append(lines, runes[start:])
	}
	return lines
}

func splitBySentences(runes []rune, maxRunes int) [][]rune {
	var segments [][]rune
	start := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		isEnd := r == '。' || r == '！' || r == '？' || r == '!' || r == '?'
		if isEnd && i+1-start > maxRunes/2 {
			segments = append(segments, runes[start:i+1])
			start = i + 1
		}
	}
	if start < len(runes) {
		remaining := runes[start:]
		// If still too long, force-split
		for len(remaining) > maxRunes {
			segments = append(segments, remaining[:maxRunes])
			remaining = remaining[maxRunes:]
		}
		if len(remaining) > 0 {
			segments = append(segments, remaining)
		}
	}
	return segments
}

// ─── LLM Prompt ──────────────────────────────────────────────────────

var systemPrompt = `你是一位专业的中文错别字检测助手。你的任务是仔细检查给定的文本，找出其中明显的错别字。

**重要原则：**
- 只标注**明显的**错别字，例如同音字误用、形近字误用等。
- **不要标注**以下内容：
  - 网文特有的口语化表达（如"俺"、"咱"、"咋"等）
  - 谐音梗、故意的错别字（如"蓝瘦香菇"）
  - 自创词、网络用语（如"yyds"、"绝绝子"等）
  - 方言表达
  - 专有名词、角色名、地名等虚构名称
  - 不确定的情况宁可不标注
- 每个错别字需要精确标注其在句子中的位置（基于 Unicode 字符计数，从 0 开始）`

func buildDetectPrompt(segment string) string {
	return fmt.Sprintf(`请检查以下文本中的错别字。以 JSON 数组格式返回结果，每个元素包含：
- "sentence": 包含错别字的完整句子原文
- "start_index": 错别字在该句子中的起始位置（Unicode 字符索引，从 0 开始）
- "end_index": 错别字在该句子中的结束位置（不含，即 [start_index, end_index) 区间）
- "error_word": 错别字原文
- "suggestion": 建议的正确写法

如果没有发现错别字，返回空数组 []。

**注意：** start_index 和 end_index 必须基于句子中的 Unicode 字符位置（中文每个字算 1 个字符），确保精确对应。

【待检查文本】
%s

请直接返回 JSON 数组，不要添加任何说明或 markdown 标记。`, segment)
}

// parseResponse extracts the JSON array from LLM response.
func parseResponse(resp string) ([]TypoSuggestion, error) {
	resp = strings.TrimSpace(resp)
	// Strip markdown code fences if present
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		var cleaned []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock || !strings.HasPrefix(strings.TrimSpace(line), "```") {
				cleaned = append(cleaned, line)
			}
		}
		resp = strings.TrimSpace(strings.Join(cleaned, "\n"))
	}

	var suggestions []TypoSuggestion
	if err := json.Unmarshal([]byte(resp), &suggestions); err != nil {
		// Try to find JSON array in the response
		start := strings.Index(resp, "[")
		end := strings.LastIndex(resp, "]")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(resp[start:end+1]), &suggestions); err2 != nil {
				return nil, fmt.Errorf("failed to parse LLM response as JSON: %w (raw: %s)", err, truncate(resp, 200))
			}
		} else {
			return nil, fmt.Errorf("no JSON array found in response (raw: %s)", truncate(resp, 200))
		}
	}

	// Validate indices
	valid := suggestions[:0]
	for _, s := range suggestions {
		sentenceLen := utf8.RuneCountInString(s.Sentence)
		if s.StartIndex >= 0 && s.EndIndex > s.StartIndex && s.EndIndex <= sentenceLen && s.ErrorWord != "" && s.Suggestion != "" {
			valid = append(valid, s)
		}
	}
	return valid, nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
