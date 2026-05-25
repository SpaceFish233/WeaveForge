package stats

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"weaveforge/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Agent struct {
	db *gorm.DB
}

func NewAgent(db *gorm.DB) *Agent {
	return &Agent{db: db}
}

// ─── Types ───────────────────────────────────────────────────────────

type DailyStatsRow struct {
	Date             string `json:"date"`
	TotalWordCount   int    `json:"total_word_count"`
	NewWords         int    `json:"new_words"`
	ChaptersModified int    `json:"chapters_modified"`
}

type WritingGoals struct {
	DailyGoal  int `json:"daily_goal"`
	WeeklyGoal int `json:"weekly_goal"`
}

// ─── Word Counting ───────────────────────────────────────────────────

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(htmlStr string) string {
	plain := htmlTagRe.ReplaceAllString(htmlStr, "")
	plain = strings.ReplaceAll(plain, "\n", " ")
	plain = strings.ReplaceAll(plain, "\r", "")
	spaceRe := regexp.MustCompile(`\s+`)
	plain = spaceRe.ReplaceAllString(plain, " ")
	return strings.TrimSpace(plain)
}

func wordCount(content string) int {
	return utf8.RuneCountInString(stripHTML(content))
}

// ─── UpdateDailyStats ────────────────────────────────────────────────

func (a *Agent) UpdateDailyStats(ctx context.Context, chapterID, oldContent, newContent string) error {
	oldCount := wordCount(oldContent)
	newCount := wordCount(newContent)
	diff := newCount - oldCount

	today := time.Now().Format("2006-01-02")

	var stat models.DailyStats
	err := a.db.WithContext(ctx).Where("date = ?", today).First(&stat).Error

	if err == gorm.ErrRecordNotFound {
		// Calculate total word count across all chapters
		totalWords, err := a.totalWordCount(ctx)
		if err != nil {
			return err
		}
		stat = models.DailyStats{
			ID:               uuid.New().String(),
			Date:             today,
			TotalWordCount:   totalWords,
			NewWords:         diff,
			ChaptersModified: 1,
		}
		if err := a.db.WithContext(ctx).Create(&stat).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update existing record
		totalWords, err := a.totalWordCount(ctx)
		if err != nil {
			return err
		}
		updates := map[string]any{
			"total_word_count":   totalWords,
			"new_words":          gorm.Expr("new_words + ?", diff),
			"chapters_modified": gorm.Expr("chapters_modified + 1"),
		}
		if err := a.db.WithContext(ctx).Model(&models.DailyStats{}).Where("id = ?", stat.ID).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

func (a *Agent) totalWordCount(ctx context.Context) (int, error) {
	var chapters []models.Chapter
	if err := a.db.WithContext(ctx).Find(&chapters).Error; err != nil {
		return 0, err
	}
	total := 0
	for _, ch := range chapters {
		total += wordCount(ch.Content)
	}
	return total, nil
}

// ─── GetStats ────────────────────────────────────────────────────────

func (a *Agent) GetStats(ctx context.Context, days int) ([]DailyStatsRow, error) {
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	var rows []models.DailyStats
	if err := a.db.WithContext(ctx).Where("date >= ?", cutoff).Order("date asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]DailyStatsRow, len(rows))
	for i, r := range rows {
		result[i] = DailyStatsRow{
			Date:             r.Date,
			TotalWordCount:   r.TotalWordCount,
			NewWords:         r.NewWords,
			ChaptersModified: r.ChaptersModified,
		}
	}
	return result, nil
}

// GetStats365 returns stats for the last 365 days (for heatmap).
func (a *Agent) GetStats365(ctx context.Context) ([]DailyStatsRow, error) {
	return a.GetStats(ctx, 365)
}

// ─── GetTodayStats ───────────────────────────────────────────────────

func (a *Agent) GetTodayStats(ctx context.Context) (*DailyStatsRow, error) {
	today := time.Now().Format("2006-01-02")
	var stat models.DailyStats
	err := a.db.WithContext(ctx).Where("date = ?", today).First(&stat).Error
	if err == gorm.ErrRecordNotFound {
		return &DailyStatsRow{
			Date:             today,
			TotalWordCount:   0,
			NewWords:         0,
			ChaptersModified: 0,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &DailyStatsRow{
		Date:             stat.Date,
		TotalWordCount:   stat.TotalWordCount,
		NewWords:         stat.NewWords,
		ChaptersModified: stat.ChaptersModified,
	}, nil
}

// GetWeekStats returns the sum of new words for the current week (Mon-Sun).
func (a *Agent) GetWeekStats(ctx context.Context) (int, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")

	var result struct {
		Total int
	}
	if err := a.db.WithContext(ctx).Model(&models.DailyStats{}).
		Where("date >= ?", weekStart).
		Select("COALESCE(SUM(new_words), 0) as total").
		Scan(&result).Error; err != nil {
		return 0, err
	}
	return result.Total, nil
}

func (a *Agent) currentStreak(ctx context.Context) (int, error) {
	var dates []string
	if err := a.db.WithContext(ctx).Model(&models.DailyStats{}).
		Where("new_words > 0").
		Order("date desc").
		Pluck("date", &dates).Error; err != nil {
		return 0, err
	}
	if len(dates) == 0 {
		return 0, nil
	}

	streak := 0
	expected := time.Now().Format("2006-01-02")
	for _, d := range dates {
		if d == expected {
			streak++
			t, _ := time.Parse("2006-01-02", expected)
			expected = t.AddDate(0, 0, -1).Format("2006-01-02")
		} else if d < expected {
			break
		}
	}
	return streak, nil
}

// GetStreak returns the current writing streak in days.
func (a *Agent) GetStreak(ctx context.Context) (int, error) {
	return a.currentStreak(ctx)
}

// ─── Goals ───────────────────────────────────────────────────────────

type GoalSettings struct {
	DailyGoal      int `json:"daily_goal"`
	WeeklyGoal     int `json:"weekly_goal"`
	StreakWarnDays int `json:"streak_warn_days"`
}

// ─── Export CSV ──────────────────────────────────────────────────────

func (a *Agent) ExportCSV(ctx context.Context) (string, error) {
	var rows []models.DailyStats
	if err := a.db.WithContext(ctx).Order("date asc").Find(&rows).Error; err != nil {
		return "", err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	exportDir := filepath.Join(home, ".weaveforge", "exports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("writing_stats_%s.csv", time.Now().Format("20060102_150405"))
	path := filepath.Join(exportDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Write UTF-8 BOM for Excel compatibility
	f.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{"日期", "总字数", "新增字数", "修改章节数"})
	for _, r := range rows {
		w.Write([]string{
			r.Date,
			fmt.Sprintf("%d", r.TotalWordCount),
			fmt.Sprintf("%d", r.NewWords),
			fmt.Sprintf("%d", r.ChaptersModified),
		})
	}

	return path, nil
}
