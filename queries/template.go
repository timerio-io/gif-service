package queries

import (
	"gif-service/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateTemplate(db *gorm.DB, countdownID string) (*models.Template, error) {
	template := &models.Template{
		ID:          uuid.New().String(),
		CountdownID: countdownID,
		DesignID:    "1",
		Name:        "Default",
		FontFamily:  "Arial",
		FontSize:    70,
		BgColor:     "#FFFFFF",
		TextColor:   "#000000",
		Layout:      "horizontal",
		ShowUnits:   true,
		CreatedAt:   time.Now(),
	}

	if err := db.Create(template).Error; err != nil {
		return nil, err
	}

	return template, nil
}

func GetTemplate(db *gorm.DB, countdownID string) (*models.Template, error) {
	var template models.Template

	if err := db.Where("countdown_id = ?", countdownID).First(&template).Error; err != nil {
		return nil, err
	}

	return &template, nil
}

func UpdateTemplate(db *gorm.DB, id string, updates *models.Template) error {
	result := db.Model(&models.Template{}).Where("id = ?", id).Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
