package foreshadow

// CandidateForeshadow is a potential foreshadow detected in text.
type CandidateForeshadow struct {
	Text        string  `json:"text"`
	Type        string  `json:"type"`
	Confidence  float64 `json:"confidence"`
	Reason      string  `json:"reason"`
	StartIndex  int     `json:"start_index"`
	EndIndex    int     `json:"end_index"`
}

// RevealSuggestion is a proposed plan to reveal a foreshadowing.
type RevealSuggestion struct {
	ForeshadowID string `json:"foreshadow_id"`
	Description  string `json:"description"`
	ForeshadowType string `json:"foreshadow_type"`
	CurrentStatus  string `json:"current_status"`
	Plans        []RevealPlan `json:"plans"`
}

// RevealPlan is a single reveal approach.
type RevealPlan struct {
	Method    string `json:"method"`    // "侧面揭示" / "直接回忆" / "事件触发"
	Paragraph string `json:"paragraph"` // draft paragraph
}

// HealthReport summarizes foreshadow lifecycle state.
type HealthReport struct {
	Total          int     `json:"total"`
	Planted        int     `json:"planted"`
	PartiallyRevealed int `json:"partially_revealed"`
	Revealed       int     `json:"revealed"`
	AvgRevealGap   float64 `json:"avg_reveal_gap"`
	StaleCount     int     `json:"stale_count"`
	StaleWarnings  []string `json:"stale_warnings"`
	PriorityCounts map[string]int `json:"priority_counts"`
}

// ForeshadowSummary for list display.
type ForeshadowSummary struct {
	ID                 string  `json:"id"`
	Description        string  `json:"description"`
	Type               string  `json:"type"`
	Status             string  `json:"status"`
	ChapterPlanted     string  `json:"chapter_planted"`
	PlannedRevealChapter int  `json:"planned_reveal_chapter"`
	RevealProgress     int     `json:"reveal_progress"`
	Priority           string  `json:"priority"`
	Confidence         float64 `json:"confidence"`
	CreatedAt          string  `json:"created_at"`
}
