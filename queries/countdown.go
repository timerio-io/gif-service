package queries

import (
	"gif-service/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateCountdown(db *gorm.DB, userId string, name string) (*models.Countdown, error) {
	countdown := &models.Countdown{
		ID:        uuid.New().String(),
		UserID:    userId,
		Name:      name,
		Type:      models.CountdownTypeEvent,
		CreatedAt: time.Now(),
	}

	if err := db.Create(countdown).Error; err != nil {
		return nil, err
	}

	return countdown, nil
}

func GetCountdownById(db *gorm.DB, id string) (*models.Countdown, error) {
	var countdown models.Countdown

	if err := db.First(&countdown, "id = ? AND is_soft_deleted = ?", id, false).Error; err != nil {
		return nil, err
	}

	return &countdown, nil
}

func ListCountdowns(db *gorm.DB, userID string, filters map[string]interface{}) ([]models.Countdown, error) {
	var countdowns []models.Countdown

	if err := db.Where("user_id = ?", userID).Where(filters).Order("created_at DESC").Find(&countdowns).Error; err != nil {
		return nil, err
	}

	return countdowns, nil
}

func DeleteCountdown(db *gorm.DB, id string) error {
	result := db.Where("id = ?", id).Delete(&models.Countdown{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func StartCountdown(db *gorm.DB, id string) error {
	now := time.Now()

	result := db.Model(&models.Countdown{}).
		Where("id = ? AND type = ? AND is_soft_deleted = ?", id, models.CountdownTypeBirthday, false).
		Update("started_at", now)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func UpdateCountdown(db *gorm.DB, id string, updates map[string]interface{}) error {
	result := db.Model(&models.Countdown{}).Where("id = ? AND is_soft_deleted = ?", id, false).Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
