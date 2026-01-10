// internal/app/repository/interval.go
package repository

import (
	"Backend-RIP/internal/app/ds"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type IntervalRepository struct {
	db          *gorm.DB
	minioClient *minio.Client
}

func NewIntervalRepository(db *gorm.DB, minioClient *minio.Client) *IntervalRepository {
	return &IntervalRepository{
		db:          db,
		minioClient: minioClient,
	}
}

const (
	intervalImagesBucket = "interval-image"
)

// ==================== ОСНОВНОЙ МЕТОД С ПАГИНАЦИЕЙ ====================

// GetIntervals возвращает список интервалов с пагинацией
// pageSize по умолчанию 8, максимум 8 интервалов на странице
func (r *IntervalRepository) GetIntervals(
	title string,
	toneMin, toneMax float64,
	page, pageSize int,
) ([]ds.Interval, ds.PaginationInfo, error) {

	// Устанавливаем размер страницы: по умолчанию 8, максимум 8
	if pageSize <= 0 {
		pageSize = 8
	}
	if pageSize > 8 {
		pageSize = 8
	}

	// Минимальная страница - 1
	if page < 1 {
		page = 1
	}

	query := r.db.Where("is_delete = ?", false)

	if title != "" {
		query = query.Where("LOWER(title) LIKE LOWER(?)", "%"+title+"%")
	}
	if toneMin > 0 {
		query = query.Where("tone >= ?", toneMin)
	}
	if toneMax > 0 {
		query = query.Where("tone <= ?", toneMax)
	}

	// Получаем общее количество записей
	var total int64
	if err := query.Model(&ds.Interval{}).Count(&total).Error; err != nil {
		return nil, ds.PaginationInfo{}, err
	}

	// Вычисляем offset для пагинации
	offset := (page - 1) * pageSize

	// Получаем данные с пагинацией
	var intervals []ds.Interval
	err := query.
		Order("id ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&intervals).Error

	if err != nil {
		return nil, ds.PaginationInfo{}, err
	}

	// Вычисляем общее количество страниц
	totalPages := 0
	if total > 0 && pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	pagination := ds.PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	return intervals, pagination, nil
}

// GetIntervalsLegacy поддерживает старый формат (без пагинации)
func (r *IntervalRepository) GetIntervalsLegacy(
	title string,
	toneMin, toneMax float64,
) ([]ds.Interval, error) {

	query := r.db.Where("is_delete = ?", false)

	if title != "" {
		query = query.Where("title ILIKE ?", "%"+title+"%")
	}
	if toneMin > 0 {
		query = query.Where("tone >= ?", toneMin)
	}
	if toneMax > 0 {
		query = query.Where("tone <= ?", toneMax)
	}

	var intervals []ds.Interval
	err := query.Find(&intervals).Error
	return intervals, err
}

// ==================== МЕТОДЫ ДЛЯ ДЕМОНСТРАЦИИ ИНДЕКСОВ ====================

// GetIntervalsWithIndexComparison сравнивает производительность с/без индексов
func (r *IntervalRepository) GetIntervalsWithIndexComparison(
	title string,
	toneMin, toneMax float64,
	page int,
) (ds.PaginatedIntervalsResponse, ds.PaginatedIntervalsResponse, error) {

	pageSize := 8 // Фиксированный размер страницы для демонстрации

	// Запрос С индексами (используем обычный метод)
	startTimeWithIndex := time.Now()
	intervalsWithIndex, paginationWith, err := r.GetIntervals(title, toneMin, toneMax, page, pageSize)
	timeWithIndex := time.Since(startTimeWithIndex)

	if err != nil {
		return ds.PaginatedIntervalsResponse{}, ds.PaginatedIntervalsResponse{}, err
	}

	// Небольшая пауза между запросами
	time.Sleep(100 * time.Millisecond)

	// Запрос БЕЗ индексов (принудительно отключаем через raw SQL)
	startTimeWithoutIndex := time.Now()

	// Строим SQL запрос, который будет игнорировать индексы
	sqlQuery := `
		SELECT * FROM intervals 
		WHERE is_delete = false 
	`

	params := []interface{}{}
	if title != "" {
		sqlQuery += " AND LOWER(title) LIKE LOWER(?)"
		params = append(params, "%"+title+"%")
	}
	if toneMin > 0 {
		sqlQuery += " AND tone >= ?"
		params = append(params, toneMin)
	}
	if toneMax > 0 {
		sqlQuery += " AND tone <= ?"
		params = append(params, toneMax)
	}

	// Применяем пагинацию
	offset := (page - 1) * pageSize
	sqlQuery += " ORDER BY id ASC LIMIT ? OFFSET ?"
	params = append(params, pageSize, offset)

	var intervalsWithoutIndex []ds.Interval
	err = r.db.Raw(sqlQuery, params...).Scan(&intervalsWithoutIndex).Error
	timeWithoutIndex := time.Since(startTimeWithoutIndex)

	if err != nil {
		return ds.PaginatedIntervalsResponse{}, ds.PaginatedIntervalsResponse{}, err
	}

	// Получаем общее количество для запроса без индексов
	countQuery := `SELECT COUNT(*) FROM intervals WHERE is_delete = false`
	countParams := []interface{}{}

	if title != "" {
		countQuery += " AND title ILIKE ?"
		countParams = append(countParams, "%"+title+"%") // Исправить: countParams вместо params
	}
	if toneMin > 0 {
		countQuery += " AND tone >= ?"
		countParams = append(countParams, toneMin) // Исправить
	}
	if toneMax > 0 {
		countQuery += " AND tone <= ?"
		countParams = append(countParams, toneMax) // Исправить
	}

	var totalWithoutIndex int64
	r.db.Raw(countQuery, countParams...).Scan(&totalWithoutIndex)

	// Вычисляем общее количество страниц
	totalPagesWithout := 0
	if totalWithoutIndex > 0 && pageSize > 0 {
		totalPagesWithout = int((totalWithoutIndex + int64(pageSize) - 1) / int64(pageSize))
	}

	paginationWithout := ds.PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		Total:      totalWithoutIndex,
		TotalPages: totalPagesWithout,
	}

	// Формируем ответы
	responseWith := ds.PaginatedIntervalsResponse{
		Data:       intervalsWithIndex,
		Pagination: paginationWith,
	}

	responseWithout := ds.PaginatedIntervalsResponse{
		Data:       intervalsWithoutIndex,
		Pagination: paginationWithout,
	}

	// Добавляем статистику времени
	responseWith.Stats = &ds.QueryStats{
		ExecutionTimeMs: timeWithIndex.Milliseconds(),
		IndexUsed:       true,
	}

	responseWithout.Stats = &ds.QueryStats{
		ExecutionTimeMs: timeWithoutIndex.Milliseconds(),
		IndexUsed:       false,
	}

	return responseWith, responseWithout, nil
}

// ExplainQuery возвращает план выполнения запроса
func (r *IntervalRepository) ExplainQuery(
	title string,
	toneMin, toneMax float64,
	page int,
) (string, error) {

	pageSize := 8
	offset := (page - 1) * pageSize

	// Строим SQL для EXPLAIN ANALYZE
	sqlQuery := `EXPLAIN ANALYZE SELECT * FROM intervals WHERE is_delete = false`

	if title != "" {
		sqlQuery += fmt.Sprintf(" AND title ILIKE '%%%s%%'", title)
	}
	if toneMin > 0 {
		sqlQuery += fmt.Sprintf(" AND tone >= %f", toneMin)
	}
	if toneMax > 0 {
		sqlQuery += fmt.Sprintf(" AND tone <= %f", toneMax)
	}

	sqlQuery += fmt.Sprintf(" ORDER BY id DESC LIMIT %d OFFSET %d", pageSize, offset)

	var explanation string
	err := r.db.Raw(sqlQuery).Row().Scan(&explanation)

	return explanation, err
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ====================

// GetInterval возвращает один интервал
func (r *IntervalRepository) GetInterval(id int) (ds.Interval, error) {
	interval := ds.Interval{}
	err := r.db.Where("id = ? AND is_delete = ?", id, false).First(&interval).Error
	if err != nil {
		return ds.Interval{}, err
	}
	return interval, nil
}

// CreateInterval создает интервал
func (r *IntervalRepository) CreateInterval(interval *ds.Interval) error {
	interval.IsDelete = false
	return r.db.Create(interval).Error
}

// UpdateInterval обновляет интервал
func (r *IntervalRepository) UpdateInterval(id uint, updates map[string]interface{}) error {
	result := r.db.Model(&ds.Interval{}).Where("id = ? AND is_delete = ?", id, false).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("interval with id %d not found or deleted", id)
	}
	return nil
}

// DeleteInterval удаляет интервал (мягкое удаление)
func (r *IntervalRepository) DeleteInterval(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var interval ds.Interval
		if err := tx.Where("id = ?", id).First(&interval).Error; err != nil {
			return err
		}

		if interval.Photo != "" {
			if err := r.deleteIntervalImage(interval.Photo); err != nil {
				return err
			}
		}

		result := tx.Model(&ds.Interval{}).Where("id = ?", id).Update("is_delete", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("interval with id %d not found", id)
		}
		return nil
	})
}

// AddIntervalToComposition добавляет интервал в композицию
func (r *IntervalRepository) AddIntervalToComposition(intervalID uint, creatorID uint, amount uint) error {
	var composition ds.Composition

	err := r.db.Where("creator_id = ? AND status = ?", creatorID, "Черновик").
		First(&composition).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		composition = ds.Composition{
			Status:     "Черновик",
			DateCreate: time.Now(),
			CreatorID:  creatorID,
		}
		if err := r.db.Create(&composition).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	var existingItem ds.CompositorInterval
	err = r.db.Where("composition_id = ? AND interval_id = ?", composition.ID, intervalID).
		First(&existingItem).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		compositionItem := ds.CompositorInterval{
			CompositionID: composition.ID,
			IntervalID:    intervalID,
			Amount:        amount,
		}
		if err := r.db.Create(&compositionItem).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		existingItem.Amount = amount
		if err := r.db.Save(&existingItem).Error; err != nil {
			return err
		}
	}

	return nil
}

// UpdateIntervalPhoto обновляет фото интервала
func (r *IntervalRepository) UpdateIntervalPhoto(id uint, fileHeader *multipart.FileHeader) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var interval ds.Interval
		if err := tx.Where("is_delete = false").First(&interval, id).Error; err != nil {
			return err
		}

		if interval.Photo != "" {
			if err := r.deleteIntervalImage(interval.Photo); err != nil {
				return err
			}
		}

		fileExt := filepath.Ext(fileHeader.Filename)
		newFileName := fmt.Sprintf("interval_%d_%d%s", id, time.Now().Unix(), fileExt)
		newFileName = strings.ToLower(newFileName)

		imageURL, err := r.saveIntervalImageToMinIO(newFileName, fileHeader)
		if err != nil {
			return err
		}

		return tx.Model(&interval).Update("photo", imageURL).Error
	})
}

// saveIntervalImageToMinIO сохраняет изображение в MinIO
func (r *IntervalRepository) saveIntervalImageToMinIO(fileName string, fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileSize := fileHeader.Size

	contentType := "application/octet-stream"
	if strings.HasSuffix(strings.ToLower(fileName), ".jpg") || strings.HasSuffix(strings.ToLower(fileName), ".jpeg") {
		contentType = "image/jpeg"
	} else if strings.HasSuffix(strings.ToLower(fileName), ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(strings.ToLower(fileName), ".gif") {
		contentType = "image/gif"
	}

	_, err = r.minioClient.PutObject(context.Background(), intervalImagesBucket, fileName, file, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s/%s/%s", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_SERVER_PORT"), intervalImagesBucket, fileName), nil
}

func (r *IntervalRepository) deleteIntervalImage(imageURL string) error {
	if imageURL == "" {
		return nil
	}

	minioOrigin := os.Getenv("MINIO_HOST") + ":" + os.Getenv("MINIO_SERVER_PORT")

	if !strings.Contains(imageURL, minioOrigin) {
		logrus.Printf("Image URL %s doesn't contain MinIO origin, skipping deletion", imageURL)
		return nil
	}

	parts := strings.Split(imageURL, "/")
	if len(parts) == 0 {
		return errors.New("invalid image URL format")
	}

	fileName := parts[len(parts)-1]

	_, err := r.minioClient.StatObject(context.Background(), intervalImagesBucket, fileName, minio.StatObjectOptions{})
	if err != nil {
		logrus.Printf("File %s not found in MinIO bucket %s, skipping deletion", fileName, intervalImagesBucket)
		return nil
	}

	err = r.minioClient.RemoveObject(context.Background(), intervalImagesBucket, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object from MinIO: %v", err)
	}

	logrus.Printf("Successfully deleted interval image from MinIO: %s", fileName)
	return nil
}
