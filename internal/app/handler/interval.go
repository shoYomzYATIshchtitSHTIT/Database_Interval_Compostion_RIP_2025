// internal/app/handler/interval.go
package handler

import (
	"Backend-RIP/internal/app/ds"
	"Backend-RIP/internal/app/middleware"
	"Backend-RIP/internal/app/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type IntervalHandler struct {
	repo *repository.Repository
}

func NewIntervalHandler(repo *repository.Repository) *IntervalHandler {
	return &IntervalHandler{
		repo: repo,
	}
}

type CreateIntervalRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Tone        float64 `json:"tone" binding:"required"`
}

type UpdateIntervalRequest struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Tone        *float64 `json:"tone"`
}

type AddIntervalToCompositionRequest struct {
	IntervalID uint `json:"interval_id" binding:"required"`
	Amount     uint `json:"amount" binding:"required,min=1"`
}

// GetIntervals godoc
// @Summary Get intervals list with pagination
// @Description Get paginated list of intervals with filtering. Always returns paginated response.
// @Tags Intervals
// @Produce json
// @Param title query string false "Filter by title"
// @Param tone_min query number false "Filter by minimum tone"
// @Param tone_max query number false "Filter by maximum tone"
// @Param page query int false "Page number (default: 1)" minimum(1) default(1)
// @Param page_size query int false "Page size (default: 8, maximum: 8)" minimum(1) maximum(8) default(8)
// @Success 200 {object} ds.PaginatedIntervalsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /intervals [get]
func (h *IntervalHandler) GetIntervals(ctx *gin.Context) {
	// Получаем параметры фильтрации
	title := ctx.Query("title")
	toneMinStr := ctx.Query("tone_min")
	toneMaxStr := ctx.Query("tone_max")

	// Получаем параметры пагинации
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "8") // Всегда пагинация

	// Конвертируем параметры фильтрации
	var toneMin, toneMax float64
	var err error

	if toneMinStr != "" {
		toneMin, err = strconv.ParseFloat(toneMinStr, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tone_min parameter"})
			return
		}
	}

	if toneMaxStr != "" {
		toneMax, err = strconv.ParseFloat(toneMaxStr, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tone_max parameter"})
			return
		}
	}

	// Конвертируем параметры пагинации
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 8
	}

	// Ограничиваем page_size максимум 8
	if pageSize > 8 {
		pageSize = 8
	}

	// Обычный запрос с пагинацией
	intervals, pagination, err := h.repo.Interval.GetIntervals(
		title, toneMin, toneMax, page, pageSize,
	)

	if err != nil {
		logrus.Error("Failed to get intervals: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get intervals"})
		return
	}

	response := ds.PaginatedIntervalsResponse{
		Data:       intervals,
		Pagination: pagination,
	}

	ctx.JSON(http.StatusOK, response)
}

// GetInterval godoc
// @Summary Get interval details
// @Description Get interval details by ID
// @Tags Intervals
// @Produce json
// @Param id path int true "Interval ID"
// @Success 200 {object} ds.Interval
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /intervals/{id} [get]
func (h *IntervalHandler) GetInterval(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval id"})
		return
	}

	interval, err := h.repo.Interval.GetInterval(id)
	if err != nil {
		logrus.Error("Failed to get interval: ", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Interval not found"})
		return
	}

	ctx.JSON(http.StatusOK, interval)
}

// CreateInterval godoc
// @Summary Create interval
// @Description Create new interval (moderator only)
// @Tags Intervals
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateIntervalRequest true "Interval data"
// @Success 201 {object} ds.Interval
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /intervals [post]
func (h *IntervalHandler) CreateInterval(ctx *gin.Context) {
	var req CreateIntervalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	interval := &ds.Interval{
		Title:       req.Title,
		Description: req.Description,
		Tone:        req.Tone,
	}

	err := h.repo.Interval.CreateInterval(interval)
	if err != nil {
		logrus.Error("Failed to create interval: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create interval"})
		return
	}

	ctx.JSON(http.StatusCreated, interval)
}

// UpdateInterval godoc
// @Summary Update interval
// @Description Update interval (moderator only)
// @Tags Intervals
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Interval ID"
// @Param request body UpdateIntervalRequest true "Update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /intervals/{id} [put]
func (h *IntervalHandler) UpdateInterval(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval id"})
		return
	}

	var req UpdateIntervalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Tone != nil {
		updates["tone"] = *req.Tone
	}

	err = h.repo.Interval.UpdateInterval(uint(id), updates)
	if err != nil {
		logrus.Error("Failed to update interval: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update interval"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Interval updated successfully"})
}

// DeleteInterval godoc
// @Summary Delete interval
// @Description Delete interval (moderator only)
// @Tags Intervals
// @Security BearerAuth
// @Param id path int true "Interval ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /intervals/{id} [delete]
func (h *IntervalHandler) DeleteInterval(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval id"})
		return
	}

	err = h.repo.Interval.DeleteInterval(uint(id))
	if err != nil {
		logrus.Error("Failed to delete interval: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete interval"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Interval deleted successfully"})
}

// AddIntervalToComposition godoc
// @Summary Add interval to composition
// @Description Add interval to draft composition
// @Tags Intervals
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body AddIntervalToCompositionRequest true "Add interval data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /intervals/add-to-composition [post]
func (h *IntervalHandler) AddIntervalToComposition(ctx *gin.Context) {
	var req AddIntervalToCompositionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Получаем ID аутентифицированного пользователя
	creatorID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	err := h.repo.Interval.AddIntervalToComposition(req.IntervalID, creatorID, req.Amount)
	if err != nil {
		logrus.Error("Failed to add interval to composition: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add interval to composition"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Interval added to composition successfully"})
}

// UpdateIntervalPhoto godoc
// @Summary Update interval photo
// @Description Update interval photo (moderator only)
// @Tags Intervals
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Interval ID"
// @Param image formData file true "Interval image"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /intervals/{id}/image [post]
func (h *IntervalHandler) UpdateIntervalPhoto(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval id"})
		return
	}

	file, err := ctx.FormFile("image")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Image file is required"})
		return
	}

	err = h.repo.Interval.UpdateIntervalPhoto(uint(id), file)
	if err != nil {
		logrus.Error("Failed to update interval photo: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update interval photo"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Interval image updated successfully"})
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ СТРУКТУРЫ И ФУНКЦИИ ====================

type ComparisonResponse struct {
	WithIndex    ds.PaginatedIntervalsResponse `json:"with_index"`
	WithoutIndex ds.PaginatedIntervalsResponse `json:"without_index"`
	Comparison   ComparisonInfo                `json:"comparison"`
	QueryPlan    string                        `json:"query_plan,omitempty"`
}

type ComparisonInfo struct {
	TimeDifferenceMs int64   `json:"time_difference_ms"`
	PerformanceGain  float64 `json:"performance_gain"`
	Recommendation   string  `json:"recommendation"`
}

func getIndexRecommendation(timeDiff int64) string {
	if timeDiff > 1000 {
		return "Индексы ускорили запрос более чем на 1 секунду. Рекомендуется использовать индексы для больших таблиц."
	} else if timeDiff > 100 {
		return "Индексы дали значительное ускорение (~" + strconv.FormatInt(timeDiff, 10) + "мс)."
	} else if timeDiff > 0 {
		return "Небольшое ускорение с индексами."
	} else if timeDiff == 0 {
		return "Разницы во времени нет. Возможно, таблица небольшая или запрос простой."
	} else {
		return "Индексы замедлили запрос. Возможно, для этого конкретного запроса индексы неэффективны."
	}
}
