package services

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"

	"weaveforge/models"

	"gorm.io/gorm"
)

type SearchResult struct {
	ChapterID    string `json:"chapter_id"`
	ChapterTitle string `json:"chapter_title"`
	Offset       int    `json:"offset"` // rune offset in plain text
	Length       int    `json:"length"` // match length in runes
	Context      string `json:"context"`
}

type ReplaceItem struct {
	ChapterID string `json:"chapter_id"`
	Offset    int    `json:"offset"`
	Length    int    `json:"length"`
}

type SearchService struct {
	db *gorm.DB
}

func NewSearchService(db *gorm.DB) *SearchService {
	return &SearchService{db: db}
}

// SearchAll searches across chapters and returns ordered results.
func (s *SearchService) SearchAll(keyword, scope string, caseSensitive, wholeWord, useRegex bool, currentChapterID, currentVolumeID string) ([]SearchResult, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, nil
	}

	// Get chapters based on scope
	chapters, err := s.getChaptersByScope(scope, currentChapterID, currentVolumeID)
	if err != nil {
		return nil, err
	}

	var allResults []SearchResult
	for _, ch := range chapters {
		plainText := stripHTML(ch.Content)
		matches := findMatches(plainText, keyword, caseSensitive, wholeWord, useRegex)
		for _, m := range matches {
			ctx := extractContext(plainText, m.offset, m.length)
			allResults = append(allResults, SearchResult{
				ChapterID:    ch.ID,
				ChapterTitle: ch.Title,
				Offset:       m.offset,
				Length:       m.length,
				Context:      ctx,
			})
		}
	}
	return allResults, nil
}

type match struct {
	offset int
	length int
}

// ReplaceInChapter replaces a single match in a chapter's HTML content.
// Returns the updated plain text content (caller should set it via UpdateChapter).
func (s *SearchService) ReplaceInChapter(chapterID string, offset, length int, replacement string) (string, error) {
	var ch models.Chapter
	if err := s.db.First(&ch, "id = ?", chapterID).Error; err != nil {
		return "", err
	}

	plainText := stripHTML(ch.Content)
	runes := []rune(plainText)
	if offset < 0 || offset+length > len(runes) {
		return "", fmt.Errorf("search: offset out of range")
	}

	// Build new plain text
	newRunes := make([]rune, 0, len(runes)+len([]rune(replacement))-length)
	newRunes = append(newRunes, runes[:offset]...)
	newRunes = append(newRunes, []rune(replacement)...)
	newRunes = append(newRunes, runes[offset+length:]...)

	return string(newRunes), nil
}

// BatchReplace replaces multiple matches across chapters.
// Returns total replacements done.
func (s *SearchService) BatchReplace(replacements []ReplaceItem, replacement string) (int, error) {
	// Group by chapter
	chapterReplacements := make(map[string][]ReplaceItem)
	for _, r := range replacements {
		chapterReplacements[r.ChapterID] = append(chapterReplacements[r.ChapterID], r)
	}

	total := 0
	for chapterID, items := range chapterReplacements {
		var ch models.Chapter
		if err := s.db.First(&ch, "id = ?", chapterID).Error; err != nil {
			continue
		}

		plainText := stripHTML(ch.Content)
		runes := []rune(plainText)

		// Sort by offset descending to avoid offset shift
		for i := 0; i < len(items)-1; i++ {
			for j := i + 1; j < len(items); j++ {
				if items[j].Offset > items[i].Offset {
					items[i], items[j] = items[j], items[i]
				}
			}
		}

		// Apply replacements from end to start
		newRunes := make([]rune, len(runes))
		copy(newRunes, runes)
		for _, item := range items {
			if item.Offset < 0 || item.Offset+item.Length > len(newRunes) {
				continue
			}
			prefix := newRunes[:item.Offset]
			suffix := newRunes[item.Offset+item.Length:]
			repl := []rune(replacement)
			combined := make([]rune, 0, len(prefix)+len(repl)+len(suffix))
			combined = append(combined, prefix...)
			combined = append(combined, repl...)
			combined = append(combined, suffix...)
			newRunes = combined
			total++
		}

		// Update chapter content (as plain text wrapped in paragraphs)
		newContent := plainTextToHTML(string(newRunes))
		if err := s.db.Model(&models.Chapter{}).Where("id = ?", chapterID).Update("content", newContent).Error; err != nil {
			return total, err
		}
	}
	return total, nil
}

func (s *SearchService) getChaptersByScope(scope, currentChapterID, currentVolumeID string) ([]models.Chapter, error) {
	var chapters []models.Chapter
	query := s.db.Order("sort_order asc")

	switch scope {
	case "chapter":
		if currentChapterID == "" {
			return nil, nil
		}
		query = query.Where("id = ?", currentChapterID)
	case "volume":
		if currentVolumeID == "" {
			return nil, nil
		}
		query = query.Where("volume_id = ?", currentVolumeID)
	default: // "all"
		// no filter
	}

	if err := query.Find(&chapters).Error; err != nil {
		return nil, err
	}
	return chapters, nil
}

// stripHTML removes HTML tags and decodes entities to get plain text.
func stripHTML(htmlStr string) string {
	// Remove tags
	re := regexp.MustCompile(`<[^>]*>`)
	plain := re.ReplaceAllString(htmlStr, "")
	// Decode HTML entities
	plain = html.UnescapeString(plain)
	// Normalize whitespace
	plain = strings.ReplaceAll(plain, "\n", " ")
	plain = strings.ReplaceAll(plain, "\r", "")
	// Collapse multiple spaces
	spaceRe := regexp.MustCompile(`\s+`)
	plain = spaceRe.ReplaceAllString(plain, " ")
	return strings.TrimSpace(plain)
}

func findMatches(text, keyword string, caseSensitive, wholeWord, useRegex bool) []match {
	var matches []match
	textRunes := []rune(text)
	keywordRunes := []rune(keyword)

	if len(keywordRunes) == 0 {
		return nil
	}

	if useRegex {
		// Regex mode
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + keyword)
		if err != nil {
			return nil
		}
		// Find all matches in rune space
		byteText := string(textRunes)
		for _, loc := range re.FindAllStringIndex(byteText, -1) {
			// Convert byte offsets to rune offsets
			runeOffset := utf8.RuneCountInString(byteText[:loc[0]])
			runeLen := utf8.RuneCountInString(byteText[loc[0]:loc[1]])
			matches = append(matches, match{offset: runeOffset, length: runeLen})
		}
		return matches
	}

	// String matching mode
	searchText := text
	searchKeyword := keyword
	if !caseSensitive {
		searchText = strings.ToLower(text)
		searchKeyword = strings.ToLower(keyword)
	}

	searchRunes := []rune(searchText)
	keyRunes := []rune(searchKeyword)

	for i := 0; i <= len(searchRunes)-len(keyRunes); i++ {
		if equalRunes(searchRunes[i:i+len(keyRunes)], keyRunes) {
			if wholeWord {
				// Check word boundaries
				if i > 0 && isWordChar(searchRunes[i-1]) {
					continue
				}
				end := i + len(keyRunes)
				if end < len(searchRunes) && isWordChar(searchRunes[end]) {
					continue
				}
			}
			matches = append(matches, match{offset: i, length: len(keyRunes)})
		}
	}
	return matches
}

func equalRunes(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r >= 0x4e00
}

func extractContext(text string, offset, length int) string {
	runes := []rune(text)
	ctxBefore := 20
	ctxAfter := 20

	start := offset - ctxBefore
	if start < 0 {
		start = 0
	}
	end := offset + length + ctxAfter
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}

// plainTextToHTML wraps plain text in <p> tags, splitting on newlines.
func plainTextToHTML(text string) string {
	if text == "" {
		return "<p></p>"
	}
	lines := strings.Split(text, "\n")
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString("<p>")
		sb.WriteString(html.EscapeString(line))
		sb.WriteString("</p>")
	}
	return sb.String()
}
