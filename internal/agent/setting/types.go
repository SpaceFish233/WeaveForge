package setting

// ConflictWarning represents a detected inconsistency between writing and world settings.
type ConflictWarning struct {
	ConflictDesc  string `json:"conflict_desc"`
	SuggestedFix  string `json:"suggested_fix"`
	ReferenceText string `json:"reference_text"`
	SettingTitle  string `json:"setting_title"`
}

// SettingInfo is a summary of a world setting for list display.
type SettingInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// jsonConflict is used internally to parse LLM JSON output.
type jsonConflict struct {
	ConflictDesc  string `json:"conflict_desc"`
	SuggestedFix  string `json:"suggested_fix"`
	ReferenceText string `json:"reference_text"`
}
