package character

import (
	"context"
	"encoding/json"

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

type CharacterSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	Race        string `json:"race"`
	Role        string `json:"role"`
	Personality string `json:"personality"`
	Status      string `json:"status"`
	Tags        string `json:"tags"`
	Avatar      string `json:"avatar"`
	CreatedAt   string `json:"created_at"`
}

func (a *Agent) CreateCharacter(ctx context.Context, c models.Character) (string, error) {
	c.ID = uuid.New().String()
	if c.Status == "" {
		c.Status = "active"
	}
	return c.ID, a.db.WithContext(ctx).Create(&c).Error
}

func (a *Agent) ListCharacters(ctx context.Context) ([]CharacterSummary, error) {
	var rows []models.Character
	if err := a.db.WithContext(ctx).Order("name asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]CharacterSummary, len(rows))
	for i, r := range rows {
		out[i] = CharacterSummary{
			ID: r.ID, Name: r.Name, Gender: r.Gender, Race: r.Race,
			Role: r.Role, Personality: r.Personality, Status: r.Status,
			Tags: r.Tags, Avatar: r.Avatar, CreatedAt: r.CreatedAt.Format("2006-01-02"),
		}
	}
	return out, nil
}

func (a *Agent) GetCharacter(ctx context.Context, id string) (*models.Character, error) {
	var c models.Character
	if err := a.db.WithContext(ctx).First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (a *Agent) UpdateCharacter(ctx context.Context, id string, upd models.Character) error {
	return a.db.WithContext(ctx).Model(&models.Character{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name": upd.Name, "gender": upd.Gender, "race": upd.Race,
		"personality": upd.Personality, "description": upd.Description,
		"first_chapter": upd.FirstChapter, "tags": upd.Tags,
		"avatar": upd.Avatar, "role": upd.Role, "status": upd.Status,
	}).Error
}

func (a *Agent) DeleteCharacter(ctx context.Context, id string) error {
	return a.db.WithContext(ctx).Delete(&models.Character{}, "id = ?", id).Error
}

// Marshal tags helper
func MarshalTags(tags []string) string {
	b, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}
	return string(b)
}
