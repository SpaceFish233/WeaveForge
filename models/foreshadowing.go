package models

import "time"

type Foreshadowing struct {
	ID                 string    `gorm:"primaryKey;size:36" json:"id"`
	Description        string    `gorm:"type:text;not null" json:"description"`
	Type               string    `gorm:"size:50;index" json:"type"`
	Status             string    `gorm:"size:30;index;default:'planted'" json:"status"`
	ChapterPlanted     string    `gorm:"size:36" json:"chapter_planted"`
	PlannedRevealChapter int    `gorm:"default:0" json:"planned_reveal_chapter"`
	RevealProgress     int       `gorm:"default:0" json:"reveal_progress"`
	RevealPlan         string    `gorm:"type:text" json:"reveal_plan"`
	Priority           string    `gorm:"size:20;default:'medium'" json:"priority"`
	Confidence         float64   `gorm:"default:0" json:"confidence"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
