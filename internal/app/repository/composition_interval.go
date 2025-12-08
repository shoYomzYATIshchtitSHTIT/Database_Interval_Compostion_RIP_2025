package repository

import (
	"Backend-RIP/internal/app/ds"
	"fmt"
	"math"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CompositionIntervalRepository struct {
	db *gorm.DB
}

func NewCompositionIntervalRepository(db *gorm.DB) *CompositionIntervalRepository {
	return &CompositionIntervalRepository{
		db: db,
	}
}

// ==================== Домен заявки (Composition) ====================

// GetCompositionCart возвращает иконку корзины (id заявки-черновика и количество интервалов)
func (r *CompositionIntervalRepository) GetCompositionCart(creatorID uint) (uint, int64, error) {
	var composition ds.Composition
	var count int64

	err := r.db.Where("creator_id = ? AND status = ?", creatorID, "Черновик").
		First(&composition).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	err = r.db.Model(&ds.CompositorInterval{}).
		Where("composition_id = ?", composition.ID).
		Count(&count).Error
	if err != nil {
		return 0, 0, err
	}

	return composition.ID, count, nil
}

// GetCompositions возвращает список заявок с фильтрацией (кроме удаленных и черновика)
func (r *CompositionIntervalRepository) GetCompositions(status string, dateFrom, dateTo time.Time, userID uint, isModerator bool) ([]ds.Composition, error) {
	var compositions []ds.Composition

	query := r.db.
		Preload("Creator", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id, login")
		}).
		Preload("Moderator", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id, login")
		}).
		Where("status != ? AND status != ?", "Удалён", "Черновик")

	if !isModerator {
		query = query.Where("creator_id = ?", userID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if !dateFrom.IsZero() {
		query = query.Where("date_create >= ?", dateFrom)
	}
	if !dateTo.IsZero() {
		query = query.Where("date_create <= ?", dateTo)
	}

	err := query.Find(&compositions).Error
	if err != nil {
		return nil, err
	}

	return compositions, nil
}

// GetComposition возвращает одну запись заявки с ее интервалами
func (r *CompositionIntervalRepository) GetComposition(id uint) (ds.Composition, error) {
	var composition ds.Composition

	err := r.db.
		Preload("CompositorIntervals.Interval").
		Preload("Creator", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id, login")
		}).
		Preload("Moderator", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id, login")
		}).
		Where("id = ?", id).
		First(&composition).Error

	if err != nil {
		return ds.Composition{}, err
	}

	return composition, nil
}

// UpdateCompositionFields обновляет поля заявки по теме
func (r *CompositionIntervalRepository) UpdateCompositionFields(id uint, updates map[string]interface{}) error {
	delete(updates, "id")
	delete(updates, "status")
	delete(updates, "creator_id")
	delete(updates, "moderator_id")
	delete(updates, "date_create")
	delete(updates, "date_finish")

	updates["date_update"] = time.Now()

	result := r.db.Model(&ds.Composition{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("composition with id %d not found", id)
	}
	return nil
}

// FormComposition формирует заявку создателем
func (r *CompositionIntervalRepository) FormComposition(id uint, creatorID uint) error {
	var composition ds.Composition
	err := r.db.Where("id = ? AND creator_id = ? AND status = ?", id, creatorID, "Черновик").
		First(&composition).Error
	if err != nil {
		return fmt.Errorf("composition not found or not in draft status")
	}

	var intervalCount int64
	err = r.db.Model(&ds.CompositorInterval{}).Where("composition_id = ?", id).Count(&intervalCount).Error
	if err != nil {
		return err
	}
	if intervalCount == 0 {
		return fmt.Errorf("at least one interval must be added to the composition")
	}

	updates := map[string]interface{}{
		"status":      "Сформирована",
		"date_update": time.Now(),
		"date_finish": gorm.Expr("NULL"),
	}

	result := r.db.Model(&ds.Composition{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("composition with id %d not found", id)
	}
	return nil
}

// CompleteComposition завершает/отклоняет заявку модератором
// repository/composition_interval.go - метод CompleteComposition
func (r *CompositionIntervalRepository) CompleteComposition(id uint, moderatorID uint, status string, calculationData map[string]interface{}) error {
	if status != "Завершена" && status != "Отклонена" {
		return fmt.Errorf("invalid status transition")
	}

	var composition ds.Composition
	err := r.db.Where("id = ? AND status = ?", id, "Сформирована").First(&composition)
	if err != nil {
		return fmt.Errorf("composition not found or not in formed status")
	}

	// ИЗМЕНЕНИЕ: НЕ вычисляем принадлежность здесь - она будет вычислена в Django
	// belonging будет пустым, Django заполнит его позже

	updates := map[string]interface{}{
		"status":       status,
		"moderator_id": moderatorID,
		"date_update":  time.Now(),
		"date_finish":  time.Now(),
		"belonging":    "", // Оставляем пустым, Django заполнит
	}

	// Добавляем расчётные данные если есть
	for key, value := range calculationData {
		updates[key] = value
	}

	result := r.db.Model(&ds.Composition{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("composition with id %d not found", id)
	}

	logrus.Infof("Composition %d completed, belonging will be calculated by Django service", id)
	return nil
}

// DeleteComposition удаляет заявку
func (r *CompositionIntervalRepository) DeleteComposition(comID uint) error {
	var composition ds.Composition
	err := r.db.Where("id = ? AND status = ?", comID, "Черновик").First(&composition).Error
	if err != nil {
		return fmt.Errorf("only draft compositions can be deleted")
	}

	updates := map[string]interface{}{
		"status":      "Удалён",
		"date_update": time.Now(),
	}

	result := r.db.Model(&ds.Composition{}).Where("id = ?", comID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("composition with id %d not found", comID)
	}
	return nil
}

// ==================== Домен м-м (CompositorInterval) ====================

// DeleteCompositionInterval удаляет интервал из заявки
func (r *CompositionIntervalRepository) DeleteCompositionInterval(compositionID uint, intervalID uint) error {
	result := r.db.Where("composition_id = ? AND interval_id = ?", compositionID, intervalID).
		Delete(&ds.CompositorInterval{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("interval not found in composition")
	}
	return nil
}

// UpdateCompositionInterval изменяет количество интервалов в заявке
func (r *CompositionIntervalRepository) UpdateCompositionInterval(compositionID uint, intervalID uint, amount uint) error {
	result := r.db.Model(&ds.CompositorInterval{}).
		Where("composition_id = ? AND interval_id = ?", compositionID, intervalID).
		Update("amount", amount)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("interval not found in composition")
	}
	return nil
}

// ==================== Вспомогательные методы ====================

// calculateClassicismCoefficient вычисляет коэффициент принадлежности к классицизму по формуле S = 1 / (1 + |μ - μ_G|)
func (r *CompositionIntervalRepository) calculateClassicismCoefficient(compositionID uint) (float64, float64) {
	var items []ds.CompositorInterval
	r.db.Preload("Interval").Where("composition_id = ?", compositionID).Find(&items)

	if len(items) == 0 {
		return 0.0, 0.0
	}

	totalTones := 0.0
	totalIntervals := 0

	for _, item := range items {
		totalTones += item.Interval.Tone * float64(item.Amount)
		totalIntervals += int(item.Amount)
	}

	if totalIntervals == 0 {
		return 0.0, 0.0
	}

	mu := totalTones / float64(totalIntervals)
	mu_G := 2.82
	S := 1.0 / (1.0 + math.Abs(mu-mu_G))

	return S, mu
}
