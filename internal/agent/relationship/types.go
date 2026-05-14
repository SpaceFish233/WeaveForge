package relationship

// RelationshipSummary is a lightweight relationship representation for API responses.
type RelationshipSummary struct {
	ID             string  `json:"id"`
	CharacterAID   string  `json:"character_a_id"`
	CharacterAName string  `json:"character_a_name"`
	CharacterBID   string  `json:"character_b_id"`
	CharacterBName string  `json:"character_b_name"`
	Type           string  `json:"type"`
	StartChapterID string  `json:"start_chapter_id"`
	EndChapterID   *string `json:"end_chapter_id"`
	Note           string  `json:"note"`
	IsActive       bool    `json:"is_active"`
}

// GraphData contains the full graph structure for frontend rendering.
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode represents a character node in the graph.
type GraphNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Role   string `json:"role"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

// GraphEdge represents a relationship edge in the graph.
type GraphEdge struct {
	ID           string  `json:"id"`
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	Type         string  `json:"type"`
	IsActive     bool    `json:"is_active"`
	StartChapter string  `json:"start_chapter"`
	EndChapter   string  `json:"end_chapter"`
	Note         string  `json:"note"`
}

// Preset relationship types with color mappings.
var PresetTypes = []struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}{
	{"恋人", "#e74c3c"},
	{"夫妻", "#c0392b"},
	{"师徒", "#3498db"},
	{"朋友", "#2ecc71"},
	{"亲情", "#e67e22"},
	{"仇敌", "#8e44ad"},
	{"陌生人", "#95a5a6"},
	{"主仆", "#1abc9c"},
}
