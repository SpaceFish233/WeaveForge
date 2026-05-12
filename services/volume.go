package services

import (
	"fmt"

	"weaveforge/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VolumeService struct {
	db *gorm.DB
}

func NewVolumeService(db *gorm.DB) *VolumeService {
	return &VolumeService{db: db}
}

func (s *VolumeService) CreateVolume(name string) (string, error) {
	id := uuid.New().String()
	var maxOrder int
	if err := s.db.Raw("SELECT COALESCE(MAX(sort_order), 0) FROM volumes").Scan(&maxOrder).Error; err != nil {
		return "", fmt.Errorf("volume: get max sort_order: %w", err)
	}

	volume := models.Volume{
		ID:        id,
		Name:      name,
		SortOrder: maxOrder + 1,
		NovelID:   "default",
	}

	if err := s.db.Create(&volume).Error; err != nil {
		return "", err
	}
	return id, nil
}

func (s *VolumeService) ListVolumes() ([]models.VolumeSummary, error) {
	var volumes []models.VolumeSummary
	err := s.db.Model(&models.Volume{}).
		Order("sort_order asc").
		Find(&volumes).Error
	return volumes, err
}

func (s *VolumeService) UpdateVolume(id, name string) error {
	return s.db.Model(&models.Volume{}).
		Where("id = ?", id).
		Update("name", name).
		Error
}

func (s *VolumeService) DeleteVolume(id string) error {
	// Clear volume_id from chapters in this volume first
	if err := s.db.Model(&models.Chapter{}).
		Where("volume_id = ?", id).
		Update("volume_id", "").Error; err != nil {
		return err
	}
	return s.db.Where("id = ?", id).Delete(&models.Volume{}).Error
}

func (s *VolumeService) GetVolume(id string) (models.Volume, error) {
	var volume models.Volume
	err := s.db.Where("id = ?", id).First(&volume).Error
	return volume, err
}
