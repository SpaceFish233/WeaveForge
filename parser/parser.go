package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ParsedChapter struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

var chapterPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^第[一二三四五六七八九十百千万零\d]+章`),
	regexp.MustCompile(`^第[一二三四五六七八九十百千万零\d]+节`),
	regexp.MustCompile(`(?i)^chapter\s+\d+`),
	regexp.MustCompile(`(?i)^section\s+\d+`),
}

var headingPattern = regexp.MustCompile(`^#{1,2}\s+.+`)
var separatorPattern = regexp.MustCompile(`^[-=]{3,}$`)

func isChapterTitle(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, p := range chapterPatterns {
		if p.MatchString(trimmed) {
			return true
		}
	}
	return false
}

func isHeading(line string) bool {
	return headingPattern.MatchString(strings.TrimSpace(line))
}

func isSeparator(line string) bool {
	return separatorPattern.MatchString(strings.TrimSpace(line))
}

func extractTitle(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !separatorPattern.MatchString(trimmed) {
			return trimmed
		}
	}
	return "未命名章节"
}

// ParseFile reads a file and splits it into chapters.
func ParseFile(filePath string) ([]ParsedChapter, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return ParseContent(string(data), filepath.Ext(filePath)), nil
}

// ParseContent parses text content and splits it into chapters.
func ParseContent(content string, ext string) []ParsedChapter {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	startIdx := 0

	// Skip YAML frontmatter for .md files
	if ext == ".md" && len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				startIdx = i + 1
				break
			}
		}
	}

	var chapters []ParsedChapter
	var currentTitle string
	var currentContent strings.Builder
	chapterCount := 0

	flush := func() {
		if chapterCount > 0 || currentContent.Len() > 0 {
			chapters = append(chapters, ParsedChapter{
				Title:   currentTitle,
				Content: strings.TrimSpace(currentContent.String()),
			})
			currentContent.Reset()
		}
	}

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if isChapterTitle(trimmed) || isHeading(trimmed) {
			flush()
			currentTitle = trimmed
			chapterCount++
		} else if isSeparator(line) {
			flush()
			currentTitle = ""
			chapterCount++
		} else {
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(line)
		}
	}

	// flush remaining content if we had any chapter markers
	if chapterCount > 0 || currentContent.Len() > 0 {
		chapters = append(chapters, ParsedChapter{
			Title:   currentTitle,
			Content: strings.TrimSpace(currentContent.String()),
		})
	}

	// Single-chapter fallback
	if len(chapters) == 0 && strings.TrimSpace(content) != "" {
		chapters = append(chapters, ParsedChapter{
			Title:   extractTitle(content),
			Content: strings.TrimSpace(content),
		})
	}

	return chapters
}
