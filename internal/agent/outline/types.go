package outline

// OutlineTreeNode is a nested tree representation for the frontend.
type OutlineTreeNode struct {
	ID           string            `json:"id"`
	ParentID     string            `json:"parent_id"`
	Title        string            `json:"title"`
	Summary      string            `json:"summary"`
	Status       string            `json:"status"`
	ChapterID    string            `json:"chapter_id"`
	ChapterTitle string            `json:"chapter_title"`
	SortOrder    int               `json:"sort_order"`
	Children     []OutlineTreeNode `json:"children"`
}
