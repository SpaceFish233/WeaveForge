package inspiration

// InspirationSummary is returned when listing inspirations.
type InspirationSummary struct {
	ID        string   `json:"id"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	Source    string   `json:"source"`
	CreatedAt string   `json:"created_at"`
}

// InspirationMatch is a matched inspiration with relevance info.
type InspirationMatch struct {
	ID        string   `json:"id"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	Score     float64  `json:"score"`
	MatchType string   `json:"match_type"` // semantic / tag
}
