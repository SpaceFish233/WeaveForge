package outline

import (
	"context"
	"fmt"
	"strings"

	"weaveforge/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Agent struct {
	db *gorm.DB
}

func NewAgent(db *gorm.DB) *Agent {
	return &Agent{db: db}
}

// ─── Tree ───────────────────────────────────────────────────────────

func (a *Agent) GetTree(ctx context.Context) ([]OutlineTreeNode, error) {
	var nodes []models.OutlineNode
	if err := a.db.WithContext(ctx).Order("sort_order asc").Find(&nodes).Error; err != nil {
		return nil, err
	}

	// Load chapter titles for binding display
	chapterTitles := make(map[string]string)
	var chapters []models.Chapter
	if err := a.db.WithContext(ctx).Find(&chapters).Error; err == nil {
		for _, ch := range chapters {
			chapterTitles[ch.ID] = ch.Title
		}
	}

	// Build tree
	return buildTree(nodes, "", chapterTitles), nil
}

func buildTree(nodes []models.OutlineNode, parentID string, chapterTitles map[string]string) []OutlineTreeNode {
	var result []OutlineTreeNode
	for _, n := range nodes {
		if n.ParentID != parentID {
			continue
		}
		node := OutlineTreeNode{
			ID:           n.ID,
			ParentID:     n.ParentID,
			Title:        n.Title,
			Summary:      n.Summary,
			Status:       n.Status,
			ChapterID:    n.ChapterID,
			ChapterTitle: chapterTitles[n.ChapterID],
			SortOrder:    n.SortOrder,
			Children:     buildTree(nodes, n.ID, chapterTitles),
		}
		result = append(result, node)
	}
	return result
}

// ─── CRUD ───────────────────────────────────────────────────────────

func (a *Agent) CreateNode(ctx context.Context, parentID, title, summary string) (string, error) {
	// Determine sort order: append to end of siblings
	var maxOrder int
	a.db.WithContext(ctx).Model(&models.OutlineNode{}).
		Where("parent_id = ?", parentID).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&maxOrder)

	n := models.OutlineNode{
		ID:        uuid.New().String(),
		ParentID:  parentID,
		Title:     title,
		Summary:   summary,
		Status:    "not_started",
		SortOrder: maxOrder + 1,
	}
	if err := a.db.WithContext(ctx).Create(&n).Error; err != nil {
		return "", err
	}
	return n.ID, nil
}

func (a *Agent) UpdateNode(ctx context.Context, id, title, summary, status string) error {
	updates := map[string]interface{}{
		"title":   title,
		"summary": summary,
	}
	if status != "" {
		updates["status"] = status
	}
	return a.db.WithContext(ctx).Model(&models.OutlineNode{}).Where("id = ?", id).Updates(updates).Error
}

func (a *Agent) DeleteNode(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return deleteNodeRecursive(tx, id)
	})
}

func deleteNodeRecursive(tx *gorm.DB, id string) error {
	// Find children
	var children []models.OutlineNode
	if err := tx.Where("parent_id = ?", id).Find(&children).Error; err != nil {
		return err
	}
	for _, child := range children {
		if err := deleteNodeRecursive(tx, child.ID); err != nil {
			return err
		}
	}
	return tx.Delete(&models.OutlineNode{}, "id = ?", id).Error
}

func (a *Agent) MoveNode(ctx context.Context, id, newParentID string, newSortOrder int) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update the moved node
		if err := tx.Model(&models.OutlineNode{}).Where("id = ?", id).Updates(map[string]interface{}{
			"parent_id":  newParentID,
			"sort_order": newSortOrder,
		}).Error; err != nil {
			return err
		}
		// Re-index siblings in new parent
		var siblings []models.OutlineNode
		if err := tx.Where("parent_id = ? AND id != ?", newParentID, id).Order("sort_order asc").Find(&siblings).Error; err != nil {
			return err
		}
		for i, s := range siblings {
			order := i + 1
			if i >= newSortOrder-1 {
				order = i + 2
			}
			if s.SortOrder != order {
				tx.Model(&models.OutlineNode{}).Where("id = ?", s.ID).Update("sort_order", order)
			}
		}
		return nil
	})
}

// ─── Chapter Binding ────────────────────────────────────────────────

func (a *Agent) BindChapter(ctx context.Context, nodeID, chapterID string) error {
	updates := map[string]interface{}{
		"chapter_id": chapterID,
	}
	// Auto-update status if chapter has content
	var ch models.Chapter
	if err := a.db.WithContext(ctx).First(&ch, "id = ?", chapterID).Error; err == nil {
		if len(ch.Content) > 0 {
			var node models.OutlineNode
			if err := a.db.WithContext(ctx).First(&node, "id = ?", nodeID).Error; err == nil {
				if node.Status == "not_started" {
					updates["status"] = "in_progress"
				}
			}
		}
	}
	return a.db.WithContext(ctx).Model(&models.OutlineNode{}).Where("id = ?", nodeID).Updates(updates).Error
}

func (a *Agent) UnbindChapter(ctx context.Context, nodeID string) error {
	return a.db.WithContext(ctx).Model(&models.OutlineNode{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
		"chapter_id": "",
	}).Error
}

// ─── Import from Chapters ───────────────────────────────────────────

func (a *Agent) ImportFromChapters(ctx context.Context) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get existing outline nodes to avoid duplicates
		var existing []models.OutlineNode
		tx.Find(&existing)
		boundChapters := make(map[string]bool)
		for _, n := range existing {
			if n.ChapterID != "" {
				boundChapters[n.ChapterID] = true
			}
		}

		// Get volumes
		var volumes []models.Volume
		if err := tx.Order("sort_order asc").Find(&volumes).Error; err != nil {
			return err
		}

		// Get chapters grouped by volume
		var chapters []models.Chapter
		if err := tx.Order("sort_order asc").Find(&chapters).Error; err != nil {
			return err
		}
		chaptersByVolume := make(map[string][]models.Chapter)
		for _, ch := range chapters {
			chaptersByVolume[ch.VolumeID] = append(chaptersByVolume[ch.VolumeID], ch)
		}

		// Track existing volume outline nodes by title to avoid duplicates
		existingVolumeNodes := make(map[string]string) // title -> nodeID
		for _, n := range existing {
			if n.ParentID == "" {
				existingVolumeNodes[n.Title] = n.ID
			}
		}

		for vi, vol := range volumes {
			volNodeID, exists := existingVolumeNodes[vol.Name]
			if !exists {
				// Create volume node
				volNode := models.OutlineNode{
					ID:        uuid.New().String(),
					ParentID:  "",
					Title:     vol.Name,
					Status:    "not_started",
					SortOrder: vi + 1,
				}
				if err := tx.Create(&volNode).Error; err != nil {
					return err
				}
				volNodeID = volNode.ID
			}

			// Create chapter nodes under this volume
			volChapters := chaptersByVolume[vol.ID]
			for ci, ch := range volChapters {
				if boundChapters[ch.ID] {
					continue // already in outline
				}
				chNode := models.OutlineNode{
					ID:        uuid.New().String(),
					ParentID:  volNodeID,
					Title:     ch.Title,
					ChapterID: ch.ID,
					Status:    "not_started",
					SortOrder: ci + 1,
				}
				if len(ch.Content) > 0 {
					chNode.Status = "in_progress"
				}
				if err := tx.Create(&chNode).Error; err != nil {
					return err
				}
			}
		}

		// Handle chapters without a volume
		orphanChapters := chaptersByVolume[""]
		if len(orphanChapters) > 0 {
			// Find or create "未分卷" node
			var orphNodeID string
			for _, n := range existing {
				if n.ParentID == "" && n.Title == "未分卷" {
					orphNodeID = n.ID
					break
				}
			}
			if orphNodeID == "" {
				orphNode := models.OutlineNode{
					ID:        uuid.New().String(),
					ParentID:  "",
					Title:     "未分卷",
					Status:    "not_started",
					SortOrder: len(volumes) + 1,
				}
				if err := tx.Create(&orphNode).Error; err != nil {
					return err
				}
				orphNodeID = orphNode.ID
			}
			for ci, ch := range orphanChapters {
				if boundChapters[ch.ID] {
					continue
				}
				chNode := models.OutlineNode{
					ID:        uuid.New().String(),
					ParentID:  orphNodeID,
					Title:     ch.Title,
					ChapterID: ch.ID,
					Status:    "not_started",
					SortOrder: ci + 1,
				}
				if len(ch.Content) > 0 {
					chNode.Status = "in_progress"
				}
				if err := tx.Create(&chNode).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// ─── Export Markdown ────────────────────────────────────────────────

func (a *Agent) ExportMarkdown(ctx context.Context) (string, error) {
	tree, err := a.GetTree(ctx)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, node := range tree {
		exportNodeMarkdown(&sb, node, 0)
	}
	return sb.String(), nil
}

func exportNodeMarkdown(sb *strings.Builder, node OutlineTreeNode, depth int) {
	indent := strings.Repeat("  ", depth)
	statusIcon := map[string]string{
		"not_started": "[ ]",
		"in_progress": "[~]",
		"completed":   "[x]",
	}
	icon := statusIcon[node.Status]
	if icon == "" {
		icon = "[ ]"
	}
	fmt.Fprintf(sb, "%s- %s %s\n", indent, icon, node.Title)
	for _, child := range node.Children {
		exportNodeMarkdown(sb, child, depth+1)
	}
}

// ─── Get node by chapter ID (for editor integration) ────────────────

func (a *Agent) GetNodeByChapterID(ctx context.Context, chapterID string) (*OutlineTreeNode, error) {
	var node models.OutlineNode
	if err := a.db.WithContext(ctx).Where("chapter_id = ?", chapterID).First(&node).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &OutlineTreeNode{
		ID:        node.ID,
		ParentID:  node.ParentID,
		Title:     node.Title,
		Summary:   node.Summary,
		Status:    node.Status,
		ChapterID: node.ChapterID,
		SortOrder: node.SortOrder,
	}, nil
}
