package services

import (
	"fmt"

	"weaveforge/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChapterService struct {
	db *gorm.DB
}

func NewChapterService(db *gorm.DB) *ChapterService {
	return &ChapterService{db: db}
}

func (s *ChapterService) CreateChapter(title, content, volumeID string) (string, error) {
	id := uuid.New().String()
	var maxOrder int
	if err := s.db.Raw("SELECT COALESCE(MAX(sort_order), 0) FROM chapters").Scan(&maxOrder).Error; err != nil {
		return "", fmt.Errorf("chapter: get max sort_order: %w", err)
	}

	chapter := models.Chapter{
		ID:        id,
		Title:     title,
		Content:   content,
		SortOrder: maxOrder + 1,
		NovelID:   "default",
		VolumeID:  volumeID,
	}

	if err := s.db.Create(&chapter).Error; err != nil {
		return "", err
	}
	return id, nil
}

func (s *ChapterService) UpdateChapter(chapterID, content string) error {
	return s.db.Model(&models.Chapter{}).
		Where("id = ?", chapterID).
		Update("content", content).
		Error
}

func (s *ChapterService) UpdateChapterTitle(chapterID, title string) error {
	return s.db.Model(&models.Chapter{}).
		Where("id = ?", chapterID).
		Update("title", title).
		Error
}

func (s *ChapterService) DeleteChapter(chapterID string) error {
	return s.db.Where("id = ?", chapterID).Delete(&models.Chapter{}).Error
}

func (s *ChapterService) GetChapter(chapterID string) (models.Chapter, error) {
	var chapter models.Chapter
	err := s.db.Where("id = ?", chapterID).First(&chapter).Error
	return chapter, err
}

func (s *ChapterService) ListChapters() ([]models.ChapterSummary, error) {
	var chapters []models.ChapterSummary
	err := s.db.Model(&models.Chapter{}).
		Order("sort_order asc").
		Find(&chapters).Error
	return chapters, err
}

func (s *ChapterService) ReorderChapters(chapterIDs []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range chapterIDs {
			if err := tx.Model(&models.Chapter{}).
				Where("id = ?", id).
				Update("sort_order", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ChapterService) UpdateChapterVolume(chapterID, volumeID string) error {
	return s.db.Model(&models.Chapter{}).
		Where("id = ?", chapterID).
		Update("volume_id", volumeID).
		Error
}
