package services

import (
	"fmt"
	"html"
	"regexp"
	"sort"
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

// htmlTextRange maps a plain-text rune range [runeOffset, runeOffset+runeLength) to
// the corresponding byte range in the original HTML source. HTML tags and entities
// are skipped when counting runes, so the returned byte positions can be used to
// splice the original HTML string without losing formatting.
//
// Returns (htmlByteStart, htmlByteEnd, ok). ok is false when the rune range is
// out of bounds.
func htmlTextRange(htmlStr string, runeOffset, runeLength int) (int, int, bool) {
	if runeOffset < 0 || runeLength < 0 {
		return 0, 0, false
	}

	var htmlStart, htmlEnd int
	htmlStart = -1
	runePos := 0
	htmlPos := 0
	remaining := htmlStr

	for len(remaining) > 0 {
		if remaining[0] == '<' {
			// Skip HTML tag
			close := strings.IndexByte(remaining, '>')
			if close < 0 {
				break
			}
			htmlPos += close + 1
			remaining = remaining[close+1:]
			continue
		}

		if remaining[0] == '&' {
			// Skip HTML entity (counts as one rune in decoded text)
			semi := strings.IndexByte(remaining, ';')
			if semi < 0 {
				// Malformed entity, treat as text
				_, sz := utf8.DecodeRuneInString(remaining)
				if runePos == runeOffset {
					htmlStart = htmlPos
				}
				runePos++
				if runePos == runeOffset+runeLength {
					htmlEnd = htmlPos + sz
					return htmlStart, htmlEnd, true
				}
				htmlPos += sz
				remaining = remaining[sz:]
				continue
			}
			if runePos == runeOffset {
				htmlStart = htmlPos
			}
			runePos++
			if runePos == runeOffset+runeLength {
				htmlEnd = htmlPos + semi + 1
				return htmlStart, htmlEnd, true
			}
			htmlPos += semi + 1
			remaining = remaining[semi+1:]
			continue
		}

		// Regular text character
		_, sz := utf8.DecodeRuneInString(remaining)
		if runePos == runeOffset {
			htmlStart = htmlPos
		}
		runePos++
		if runePos == runeOffset+runeLength {
			htmlEnd = htmlPos + sz
			return htmlStart, htmlEnd, true
		}
		htmlPos += sz
		remaining = remaining[sz:]
	}

	// If we matched up to the end
	if htmlStart >= 0 && runePos >= runeOffset+runeLength {
		htmlEnd = htmlPos
		return htmlStart, htmlEnd, true
	}

	return 0, 0, false
}

// ReplaceInChapter replaces a single match in a chapter's HTML content.
// It operates directly on the HTML source, preserving all formatting tags.
func (s *SearchService) ReplaceInChapter(chapterID string, offset, length int, replacement string) (string, error) {
	var ch models.Chapter
	if err := s.db.First(&ch, "id = ?", chapterID).Error; err != nil {
		return "", err
	}

	htmlStart, htmlEnd, ok := htmlTextRange(ch.Content, offset, length)
	if !ok || htmlStart < 0 {
		return "", fmt.Errorf("search: offset out of range")
	}

	htmlBytes := []byte(ch.Content)
	escaped := html.EscapeString(replacement)
	result := make([]byte, 0, len(htmlBytes)+len(escaped)-length)
	result = append(result, htmlBytes[:htmlStart]...)
	result = append(result, []byte(escaped)...)
	result = append(result, htmlBytes[htmlEnd:]...)

	return string(result), nil
}

// BatchReplace replaces multiple matches across chapters in the HTML source.
// Replacements operate directly on HTML, preserving all formatting.
// Returns total replacements done.
func (s *SearchService) BatchReplace(replacements []ReplaceItem, replacement string) (int, error) {
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

		// Validate no overlapping replacements
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				a, b := items[i], items[j]
				if a.Offset < b.Offset+b.Length && b.Offset < a.Offset+a.Length {
					return total, fmt.Errorf("overlapping replacements in chapter %s", chapterID)
				}
			}
		}

		// Sort by offset descending to apply from end to start
		sort.Slice(items, func(i, j int) bool {
			return items[i].Offset > items[j].Offset
		})

		htmlContent := ch.Content
		for _, item := range items {
			htmlStart, htmlEnd, ok := htmlTextRange(htmlContent, item.Offset, item.Length)
			if !ok || htmlStart < 0 || htmlEnd < htmlStart {
				continue
			}
			escaped := html.EscapeString(replacement)
			htmlContent = htmlContent[:htmlStart] + escaped + htmlContent[htmlEnd:]
			total++
		}

		if err := s.db.Model(&models.Chapter{}).Where("id = ?", chapterID).Update("content", htmlContent).Error; err != nil {
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
	}

	if err := query.Find(&chapters).Error; err != nil {
		return nil, err
	}
	return chapters, nil
}

// stripHTML removes HTML tags and decodes entities to get plain text.
// Used only for search (finding match positions), not for replacement.
func stripHTML(htmlStr string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	plain := re.ReplaceAllString(htmlStr, "")
	plain = html.UnescapeString(plain)
	plain = strings.ReplaceAll(plain, "\n", " ")
	plain = strings.ReplaceAll(plain, "\r", "")
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
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + keyword)
		if err != nil {
			return nil
		}
		byteText := string(textRunes)
		for _, loc := range re.FindAllStringIndex(byteText, -1) {
			runeOffset := utf8.RuneCountInString(byteText[:loc[0]])
			runeLen := utf8.RuneCountInString(byteText[loc[0]:loc[1]])
			matches = append(matches, match{offset: runeOffset, length: runeLen})
		}
		return matches
	}

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
