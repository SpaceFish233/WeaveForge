package models

import "time"

type StyleProfile struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	Features    string    `gorm:"type:text" json:"features"`       // JSON of StyleFeatures
	Samples     string    `gorm:"type:text" json:"samples"`        // JSON of sample paragraphs
	Description string    `gorm:"type:text" json:"description"`    // Human-readable style summary
	SourceChapters string `gorm:"type:text" json:"source_chapters"` // JSON array of chapter IDs
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
