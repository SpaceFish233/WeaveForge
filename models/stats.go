package models

import "time"

type DailyStats struct {
	ID               string    `gorm:"primaryKey;size:36" json:"id"`
	Date             string    `gorm:"size:10;uniqueIndex;not null" json:"date"` // YYYY-MM-DD
	TotalWordCount   int       `json:"total_word_count"`
	NewWords         int       `json:"new_words"`
	ChaptersModified int       `json:"chapters_modified"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
