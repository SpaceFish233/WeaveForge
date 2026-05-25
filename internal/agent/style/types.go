package style

// StyleFeatures contains quantitative writing style metrics.
type StyleFeatures struct {
	AvgSentenceLength  float64            `json:"avg_sentence_length"`
	AvgParagraphLength float64            `json:"avg_paragraph_length"`
	SentenceLengthStd  float64            `json:"sentence_length_std"`
	DialogueRatio      float64            `json:"dialogue_ratio"`
	VocabularyRichness float64            `json:"vocabulary_richness"`
	TotalChars         int                `json:"total_chars"`
	TotalSentences     int                `json:"total_sentences"`
	TotalParagraphs    int                `json:"total_paragraphs"`
	TopChars           []CharFreq         `json:"top_chars"`
	TopBigrams         []CharFreq         `json:"top_bigrams"`
	PunctuationFreq    map[string]float64 `json:"punctuation_freq"`
	RhetoricDensity    float64            `json:"rhetoric_density"`
}

type CharFreq struct {
	Char string  `json:"char"`
	Freq float64 `json:"freq"`
}

// StyleProfileSummary is returned when listing profiles.
type StyleProfileSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// PolishRequest is the input for a polish operation.
type PolishRequest struct {
	Text       string `json:"text"`
	ProfileID  string `json:"profile_id"`
	Intensity  string `json:"intensity"` // 轻微 / 中等 / 较大
}

// PolishResult is returned from a polish operation.
type PolishResult struct {
	Original string `json:"original"`
	Polished string `json:"polished"`
}

// ─── AI Flavor Detection ──────────────────────────────────────────────

// AIFlavorIssue represents a single detected AI-flavor problem.
type AIFlavorIssue struct {
	Description string `json:"description"`
	Evidence    string `json:"evidence"`
	FixHint     string `json:"fix_hint"`
}

// AIFlavorDimension is one of the five Anti-AI check dimensions.
type AIFlavorDimension struct {
	Label    string          `json:"label"`
	Severity string          `json:"severity"` // pass / low / medium / high / critical
	Issues   []AIFlavorIssue `json:"issues"`
}

// AIFlavorReport is the full five-dimension Anti-AI flavor detection result.
type AIFlavorReport struct {
	Dimensions []AIFlavorDimension `json:"dimensions"`
	Summary    string              `json:"summary"`
	CheckedAt  string              `json:"checked_at"`
}

// ─── Chapter Hook Check ───────────────────────────────────────────────

// HookCheckResult analyzes the chapter ending for hook quality.
type HookCheckResult struct {
	HasHook             bool     `json:"has_hook"`
	HookType            string   `json:"hook_type"`
	HookStrength        string   `json:"hook_strength"`
	UnresolvedQuestions []string `json:"unresolved_questions"`
	ClosingAnalysis     string   `json:"closing_analysis"`
	Suggestion          string   `json:"suggestion"`
	CheckedAt           string   `json:"checked_at"`
}

// ─── Pre-Write Constraint Check ──────────────────────────────────────

// ConstraintIssue represents a single constraint violation.
type ConstraintIssue struct {
	Category    string `json:"category"` // setting / character / timeline / logic / continuity
	Severity    string `json:"severity"` // critical / high / medium / low
	Description string `json:"description"`
	Evidence    string `json:"evidence"`
	FixHint     string `json:"fix_hint"`
}

// ConstraintCheckResult is the full pre-write constraint check report.
type ConstraintCheckResult struct {
	Passed        bool              `json:"passed"`
	Issues        []ConstraintIssue `json:"issues"`
	BlockingCount int               `json:"blocking_count"`
	Summary       string            `json:"summary"`
	CheckedAt     string            `json:"checked_at"`
}

// ─── Placeholder Scan ────────────────────────────────────────────────

// PlaceholderMatch represents a single placeholder found in text.
type PlaceholderMatch struct {
	Pattern    string `json:"pattern"`
	Type       string `json:"type"` // todo / temp_name / placeholder / ellipsis
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	Context    string `json:"context"`
}

// PlaceholderScanResult is the result of scanning text for unfinished markers.
type PlaceholderScanResult struct {
	Clean     bool              `json:"clean"`
	Matches   []PlaceholderMatch `json:"matches"`
	CheckedAt string            `json:"checked_at"`
}
