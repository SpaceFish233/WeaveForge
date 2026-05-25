package timeline

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var timePatterns = []struct {
	pattern *regexp.Regexp
	handler func(match []string, content string) extractedEvent
}{
	{
		regexp.MustCompile(`(星历|纪元|历)\s*(\d{2,6})\s*年\s*(\d{1,2})?\s*月?\s*(\d{1,2})?\s*日?`),
		func(m []string, _ string) extractedEvent {
			year := m[2]
			label := m[1] + year + "年"
			if m[3] != "" {
				label += m[3] + "月"
			}
			if m[4] != "" {
				label += m[4] + "日"
			}
			return extractedEvent{
				Label:       label,
				Title:       label,
				RawTimeExpr: m[0],
				Offset:      parseYearToDays(year),
			}
		},
	},
	{
		regexp.MustCompile(`(\d{2,6})\s*年\s*(\d{1,2})\s*月\s*(\d{1,2})\s*[日号]`),
		func(m []string, _ string) extractedEvent {
			label := m[0]
			return extractedEvent{
				Label:       label,
				Title:       label,
				RawTimeExpr: m[0],
				Offset:      parseYearToDays(m[1]),
			}
		},
	},
	{
		regexp.MustCompile(`(第[一二三四五六七八九十百千万\d]+天|翌日|次日|当日|当天|隔天|第二天|第三天|第四天|第五天)`),
		func(m []string, _ string) extractedEvent {
			days := parseRelativeDays(m[1])
			return extractedEvent{
				Label:       m[1],
				Title:       m[1],
				RawTimeExpr: m[0],
				Offset:      days,
			}
		},
	},
	{
		regexp.MustCompile(`(\d+)\s*[天日]\s*后`),
		func(m []string, _ string) extractedEvent {
			var days int
			fmt.Sscanf(m[1], "%d", &days)
			label := fmt.Sprintf("%d天后", days)
			return extractedEvent{
				Label:       label,
				Title:       label,
				RawTimeExpr: m[0],
				Offset:      float64(days),
			}
		},
	},
	{
		regexp.MustCompile(`(\d+)\s*个?\s*月\s*后`),
		func(m []string, _ string) extractedEvent {
			var months int
			fmt.Sscanf(m[1], "%d", &months)
			label := fmt.Sprintf("%d个月后", months)
			return extractedEvent{
				Label:       label,
				Title:       label,
				RawTimeExpr: m[0],
				Offset:      float64(months) * 30,
			}
		},
	},
	{
		regexp.MustCompile(`(\d+)\s*年\s*后`),
		func(m []string, _ string) extractedEvent {
			var years int
			fmt.Sscanf(m[1], "%d", &years)
			label := fmt.Sprintf("%d年后", years)
			return extractedEvent{
				Label:       label,
				Title:       label,
				RawTimeExpr: m[0],
				Offset:      float64(years) * 365,
			}
		},
	},
	{
		regexp.MustCompile(`(一个|两个|三个|四个|五个|六|七|八|九|十|\d+)\s*时辰\s*后`),
		func(m []string, _ string) extractedEvent {
			hours := parseChineseNum(m[1])
			label := fmt.Sprintf("%s时辰后", m[1])
			return extractedEvent{
				Label:       label,
				Title:       label,
				RawTimeExpr: m[0],
				Offset:      float64(hours) / 12,
			}
		},
	},
	{
		regexp.MustCompile(`(一炷香|半柱香|一盏茶|一弹指)\s*后?`),
		func(m []string, _ string) extractedEvent {
			return extractedEvent{
				Label:       m[1],
				Title:       m[1],
				RawTimeExpr: m[0],
				Offset:      0.01, // very short, same day
			}
		},
	},
	{
		regexp.MustCompile(`(大[秦汉唐宋元明清]\S{0,10}(?:年|岁|纪))\s*(\S{0,5}(?:春|夏|秋|冬))?`),
		func(m []string, _ string) extractedEvent {
			label := strings.TrimSpace(m[0])
			return extractedEvent{
				Label:       label,
				Title:       label,
				RawTimeExpr: m[0],
				Offset:      -1, // needs manual positioning
			}
		},
	},
}

// extractByRegex extracts time expressions from chapter content using regex patterns.
func extractByRegex(content, chapterID string) []extractedEvent {
	var events []extractedEvent
	seen := make(map[string]bool)

	for _, tp := range timePatterns {
		matches := tp.pattern.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			ev := tp.handler(m, content)
			ev.ChapterIDs = []string{chapterID}
			// Deduplicate by raw expression
			key := ev.RawTimeExpr
			if seen[key] {
				continue
			}
			seen[key] = true

			// Extract surrounding context as summary
			ev.Summary = extractContext(content, ev.RawTimeExpr, 50)
			events = append(events, ev)
		}
	}

	return events
}

// extractContext gets surrounding text around a match.
func extractContext(content, match string, radius int) string {
	idx := strings.Index(content, match)
	if idx < 0 {
		return ""
	}
	runes := []rune(content)
	runeIdx := utf8.RuneCountInString(content[:idx])
	matchRuneLen := utf8.RuneCountInString(match)

	start := runeIdx - radius
	if start < 0 {
		start = 0
	}
	end := runeIdx + matchRuneLen + radius
	if end > len(runes) {
		end = len(runes)
	}
	ctx := string(runes[start:end])
	ctx = strings.ReplaceAll(ctx, "\n", " ")
	return ctx
}

func parseYearToDays(yearStr string) float64 {
	var year int
	fmt.Sscanf(yearStr, "%d", &year)
	return float64(year) * 365
}

func parseRelativeDays(s string) float64 {
	switch s {
	case "翌日", "次日", "第二天":
		return 1
	case "当日", "当天", "隔天":
		return 0
	case "第三天":
		return 2
	case "第四天":
		return 3
	case "第五天":
		return 4
	default:
		// Parse "第X天"
		s = strings.TrimPrefix(s, "第")
		s = strings.TrimSuffix(s, "天")
		n := parseChineseNum(s)
		if n > 0 {
			return float64(n - 1)
		}
		return 0
	}
}

func parseChineseNum(s string) int {
	s = strings.TrimSpace(s)
	// Try direct number
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
		return n
	}
	// Simple Chinese number mapping
	multiplier := 1
	result := 0
	for _, r := range s {
		switch r {
		case '一':
			result += 1 * multiplier
		case '两', '二':
			result += 2 * multiplier
		case '三':
			result += 3 * multiplier
		case '四':
			result += 4 * multiplier
		case '五':
			result += 5 * multiplier
		case '六':
			result += 6 * multiplier
		case '七':
			result += 7 * multiplier
		case '八':
			result += 8 * multiplier
		case '九':
			result += 9 * multiplier
		case '十':
			if result == 0 {
				result = 1
			}
			result *= 10
			multiplier = 1
		}
	}
	return result
}
