package db

import (
	"os"
	"path/filepath"

	"weaveforge/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dbDir := filepath.Join(home, ".weaveforge")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(dbDir, "weaveforge.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}

	if err := db.AutoMigrate(&models.Volume{}, &models.Chapter{}, &models.WorldSetting{}, &models.StyleProfile{}, &models.Foreshadowing{}, &models.Character{}); err != nil {
		return err
	}

	DB = db
	return nil
}
