package models

import "time"

type Inspiration struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	Tags      string    `gorm:"type:text" json:"tags"`            // JSON string array
	Source    string    `gorm:"size:50;default:'manual'" json:"source"` // manual / deprecated
	SourceRef string    `gorm:"size:255" json:"source_ref"`       // chapter ID or other reference
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
