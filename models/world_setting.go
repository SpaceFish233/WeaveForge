package models

import "time"

type WorldSetting struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Title     string    `gorm:"size:255;not null;index" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	Type      string    `gorm:"size:50;index" json:"type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
