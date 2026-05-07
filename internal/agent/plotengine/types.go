package plotengine

// Branch is a generated story direction.
type Branch struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Summary       string        `json:"summary"`
	PlotPoints    []PlotPoint   `json:"plot_points"`
	RevealDetails []RevealDetail `json:"reveal_details"`
}

// PlotPoint is a single story beat within a branch.
type PlotPoint struct {
	Description string `json:"description"`
	Order       int    `json:"order"`
	Type        string `json:"type"` // 事件/转折/冲突/情感
}

// RevealDetail describes how a foreshadow is resolved in a branch.
type RevealDetail struct {
	ForeshadowID   string `json:"foreshadow_id"`
	ForeshadowDesc string `json:"foreshadow_desc"`
	HowRevealed    string `json:"how_revealed"`
}

// BranchAnalysis contains LLM critique of a branch.
type BranchAnalysis struct {
	BranchID         string            `json:"branch_id"`
	LogicIssues      []string          `json:"logic_issues"`
	RhythmNotes      string            `json:"rhythm_notes"`
	ExpectationCurve []ExpectationPoint `json:"expectation_curve"`
	ReaderTags       map[string][]string `json:"reader_tags"`
	OverallRating    int               `json:"overall_rating"`
}

// ExpectationPoint is a single point on the excitement curve.
type ExpectationPoint struct {
	Position string `json:"position"` // 开篇/中段/结尾
	Score    int    `json:"score"`    // 1-10
}

// GenerationParams bundles inputs for branch generation.
type GenerationParams struct {
	ChapterSummary string   `json:"chapter_summary"`
	BranchCount    int      `json:"branch_count"`
	ForeshadowIDs  []string `json:"foreshadow_ids"`
}
