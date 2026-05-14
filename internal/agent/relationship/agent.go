package relationship

import (
	"context"
	"fmt"

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

// ─── CRUD ───────────────────────────────────────────────────────────

func (a *Agent) GetRelationships(ctx context.Context, filterChapterID string) ([]RelationshipSummary, error) {
	var rels []models.Relationship
	query := a.db.WithContext(ctx)

	if filterChapterID != "" {
		// Filter: start_chapter <= filter AND (end_chapter IS NULL OR end_chapter > filter)
		// We need to compare by chapter sort_order, so use subquery
		query = query.Where(
			"(start_chapter_id IN (SELECT id FROM chapters WHERE sort_order <= (SELECT sort_order FROM chapters WHERE id = ?)) "+
				"AND (end_chapter_id IS NULL OR end_chapter_id IN (SELECT id FROM chapters WHERE sort_order > (SELECT sort_order FROM chapters WHERE id = ?))))",
			filterChapterID, filterChapterID,
		)
	}

	if err := query.Order("created_at desc").Find(&rels).Error; err != nil {
		return nil, err
	}

	// Load character names
	charMap, _ := a.loadCharacterMap(ctx)

	result := make([]RelationshipSummary, len(rels))
	for i, r := range rels {
		result[i] = RelationshipSummary{
			ID:             r.ID,
			CharacterAID:   r.CharacterAID,
			CharacterAName: charMap[r.CharacterAID],
			CharacterBID:   r.CharacterBID,
			CharacterBName: charMap[r.CharacterBID],
			Type:           r.Type,
			StartChapterID: r.StartChapterID,
			EndChapterID:   r.EndChapterID,
			Note:           r.Note,
			IsActive:       r.EndChapterID == nil,
		}
	}
	return result, nil
}

func (a *Agent) CreateRelationship(ctx context.Context, charAID, charBID, relType, startChapterID, note string) (string, error) {
	if charAID == charBID {
		return "", fmt.Errorf("relationship: cannot relate a character to itself")
	}
	if relType == "" {
		return "", fmt.Errorf("relationship: type is required")
	}

	r := models.Relationship{
		ID:             uuid.New().String(),
		CharacterAID:   charAID,
		CharacterBID:   charBID,
		Type:           relType,
		StartChapterID: startChapterID,
		EndChapterID:   nil,
		Note:           note,
	}
	if err := a.db.WithContext(ctx).Create(&r).Error; err != nil {
		return "", err
	}
	return r.ID, nil
}

func (a *Agent) UpdateRelationship(ctx context.Context, id, relType, startChapterID, endChapterID, note string) error {
	updates := map[string]interface{}{
		"type":             relType,
		"start_chapter_id": startChapterID,
		"note":             note,
	}
	if endChapterID != "" {
		updates["end_chapter_id"] = endChapterID
	} else {
		updates["end_chapter_id"] = nil
	}
	return a.db.WithContext(ctx).Model(&models.Relationship{}).Where("id = ?", id).Updates(updates).Error
}

func (a *Agent) DeleteRelationship(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Delete(&models.Relationship{}, "id = ?", id).Error
}

// ─── Graph Data ─────────────────────────────────────────────────────

func (a *Agent) GetGraphData(ctx context.Context, filterChapterID string) (*GraphData, error) {
	// Get relationships
	rels, err := a.GetRelationships(ctx, filterChapterID)
	if err != nil {
		return nil, err
	}

	// Get all characters
	var chars []models.Character
	if err := a.db.WithContext(ctx).Find(&chars).Error; err != nil {
		return nil, err
	}

	// Create nodes for all characters
	nodes := make([]GraphNode, len(chars))
	for i, c := range chars {
		nodes[i] = GraphNode{
			ID:     c.ID,
			Name:   c.Name,
			Avatar: c.Avatar,
			Role:   c.Role,
			X:      c.GraphX,
			Y:      c.GraphY,
		}
	}

	// Create edges
	edges := make([]GraphEdge, len(rels))
	for i, r := range rels {
		endCh := ""
		if r.EndChapterID != nil {
			endCh = *r.EndChapterID
		}
		edges[i] = GraphEdge{
			ID:           r.ID,
			Source:       r.CharacterAID,
			Target:       r.CharacterBID,
			Type:         r.Type,
			IsActive:     r.IsActive,
			StartChapter: r.StartChapterID,
			EndChapter:   endCh,
			Note:         r.Note,
		}
	}

	return &GraphData{Nodes: nodes, Edges: edges}, nil
}

// ─── Node Positions ─────────────────────────────────────────────────

type NodePosition struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

func (a *Agent) SaveNodePositions(ctx context.Context, positions []NodePosition) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, p := range positions {
			if err := tx.Model(&models.Character{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
				"graph_x": p.X,
				"graph_y": p.Y,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ─── Helpers ────────────────────────────────────────────────────────

func (a *Agent) loadCharacterMap(ctx context.Context) (map[string]string, error) {
	var chars []models.Character
	if err := a.db.WithContext(ctx).Find(&chars).Error; err != nil {
		return nil, err
	}
	m := make(map[string]string, len(chars))
	for _, c := range chars {
		m[c.ID] = c.Name
	}
	return m, nil
}
