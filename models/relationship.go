package models

import "time"

// Relationship represents a relationship between two characters.
type Relationship struct {
	ID            string  `gorm:"primaryKey;size:36" json:"id"`
	CharacterAID  string  `gorm:"size:36;index;not null" json:"character_a_id"`
	CharacterBID  string  `gorm:"size:36;index;not null" json:"character_b_id"`
	Type          string  `gorm:"size:50;not null" json:"type"`
	StartChapterID string `gorm:"size:36;index" json:"start_chapter_id"`
	EndChapterID  *string `gorm:"size:36;index" json:"end_chapter_id"` // nullable = ongoing
	Note          string  `gorm:"type:text" json:"note"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
