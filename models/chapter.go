package models

import "time"

type Chapter struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	NovelID   string    `gorm:"size:36;default:'default'" json:"novel_id"`
	VolumeID  string    `gorm:"size:36;index" json:"volume_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChapterSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	SortOrder int       `json:"sort_order"`
	VolumeID  string    `json:"volume_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
