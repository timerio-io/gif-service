package queries

import (
	"gif-service/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreatePalette(db *gorm.DB, userID, name, primaryColor, secondaryColor, accentColor string) (*models.ColorPalette, error) {
	palette := &models.ColorPalette{
		ID:             uuid.New().String(),
		UserID:         userID,
		Name:           name,
		PrimaryColor:   primaryColor,
		SecondaryColor: secondaryColor,
		AccentColor:    accentColor,
		CreatedAt:      time.Now(),
	}

	if err := db.Create(palette).Error; err != nil {
		return nil, err
	}

	return palette, nil
}

func ListPalettes(db *gorm.DB, userID string) ([]models.ColorPalette, error) {
	var palettes []models.ColorPalette

	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&palettes).Error; err != nil {
		return nil, err
	}

	return palettes, nil
}

func UpdatePalette(db *gorm.DB, id, userID string, updates map[string]interface{}) error {
	result := db.Model(&models.ColorPalette{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func DeletePalette(db *gorm.DB, id, userID string) error {
	result := db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.ColorPalette{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func CountPalettes(db *gorm.DB, userID string) (int64, error) {
	var count int64

	if err := db.Model(&models.ColorPalette{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
