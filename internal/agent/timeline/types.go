package timeline

// NodeWithEvents is a timeline node with its attached events for API responses.
type NodeWithEvents struct {
	ID         string           `json:"id"`
	OffsetDays float64          `json:"offset_days"`
	Label      string           `json:"label"`
	Description string          `json:"description"`
	Events     []EventSummary   `json:"events"`
}

// EventSummary is a lightweight event representation.
type EventSummary struct {
	ID          string   `json:"id"`
	NodeID      string   `json:"node_id"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	ChapterIDs  []string `json:"chapter_ids"`
	IsGradual   bool     `json:"is_gradual"`
	EventName   string   `json:"event_name"`
	RawTimeExpr string   `json:"raw_time_expr"`
	SortOrder   int      `json:"sort_order"`
}

// TimeExpression holds an extracted time reference from text.
type TimeExpression struct {
	Raw      string  `json:"raw"`       // original text
	Offset   float64 `json:"offset"`    // days from time base, -1 if unknown
	IsAbs    bool    `json:"is_abs"`    // true if absolute time
	AbsLabel string  `json:"abs_label"` // human-readable absolute label
}

// ScanResult is returned after scanning chapters.
type ScanResult struct {
	Nodes    []NodeWithEvents `json:"nodes"`
	Conflicts []ConflictInfo   `json:"conflicts"`
}

// ConflictInfo describes a detected time conflict.
type ConflictInfo struct {
	Description string   `json:"description"`
	ChapterIDs  []string `json:"chapter_ids"`
	EventTitles []string `json:"event_titles"`
}
