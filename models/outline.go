package models

import "time"

type OutlineNode struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	ParentID  string    `gorm:"size:36;index" json:"parent_id"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Summary   string    `gorm:"type:text" json:"summary"`
	Status    string    `gorm:"size:20;default:'not_started'" json:"status"` // not_started / in_progress / completed
	ChapterID string    `gorm:"size:36;index" json:"chapter_id"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
