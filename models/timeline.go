package models

import "time"

// TimeBase stores the story's absolute time origin.
type TimeBase struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Description string    `gorm:"type:text" json:"description"` // e.g. "星历 2000 年 1 月 1 日"
	Unit        string    `gorm:"size:20" json:"unit"`          // 年/月/日/时辰
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TimelineNode represents a point on the timeline.
type TimelineNode struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	OffsetDays  float64   `gorm:"index" json:"offset_days"` // days from TimeBase
	Label       string    `gorm:"size:255" json:"label"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TimelineEvent represents a story event attached to a timeline node.
type TimelineEvent struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	NodeID      string    `gorm:"size:36;index" json:"node_id"`
	Title       string    `gorm:"size:255" json:"title"`
	Summary     string    `gorm:"type:text" json:"summary"`
	ChapterIDs  string    `gorm:"type:text" json:"chapter_ids"`  // JSON array
	IsGradual   bool      `gorm:"default:false" json:"is_gradual"`
	EventName   string    `gorm:"size:255;index" json:"event_name"` // clustering key
	RawTimeExpr string    `gorm:"size:255" json:"raw_time_expr"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
