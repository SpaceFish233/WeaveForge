package setting

// SettingHit represents a detected occurrence of a setting's title in chapter content.
type SettingHit struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Occurrences  int    `json:"occurrences"`
	FirstSnippet string `json:"first_snippet"`
}

// ConflictResult is the AI analysis result for a single setting validated against chapter content.
type ConflictResult struct {
	SettingTitle  string   `json:"setting_title"`
	HasConflict   bool     `json:"has_conflict"`
	ConflictDesc  string   `json:"conflict_desc"`
	SuggestedFix  string   `json:"suggested_fix"`
	ReferenceText string   `json:"reference_text"`
	Snippets      []string `json:"snippets"`
	TruncatedFrom int      `json:"truncated_from"`
}

// SettingInfo is a summary of a world setting for list display.
type SettingInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// jsonConflict is used internally to parse LLM JSON output.
type jsonConflict struct {
	HasConflict   bool   `json:"has_conflict"`
	ConflictDesc  string `json:"conflict_desc"`
	SuggestedFix  string `json:"suggested_fix"`
	ReferenceText string `json:"reference_text"`
}
