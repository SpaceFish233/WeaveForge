package models

import "time"

type Character struct {
	ID            string    `gorm:"primaryKey;size:36" json:"id"`
	Name          string    `gorm:"size:100;not null;index" json:"name"`
	Gender        string    `gorm:"size:20" json:"gender"`
	Race          string    `gorm:"size:50" json:"race"`
	Personality   string    `gorm:"size:255" json:"personality"`
	Description   string    `gorm:"type:text" json:"description"`
	FirstChapter  string    `gorm:"size:36" json:"first_chapter"`
	Tags          string    `gorm:"size:255" json:"tags"` // JSON array
	Avatar        string    `gorm:"size:255" json:"avatar"` // placeholder / URL
	Role          string    `gorm:"size:50" json:"role"` // 主角 / 配角 / 反派 / 路人
	Status        string    `gorm:"size:30;default:'active'" json:"status"` // active / inactive / deceased
	GraphX        float64   `gorm:"default:0" json:"graph_x"`
	GraphY        float64   `gorm:"default:0" json:"graph_y"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
