package models

import "time"

type Volume struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	NovelID   string    `gorm:"size:36;default:'default'" json:"novel_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VolumeSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}
