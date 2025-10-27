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
// @Summary Get intervals list
// @Description Get list of intervals with filtering
// @Tags Intervals
// @Produce json
// @Param title query string false "Filter by title"
// @Param tone_min query number false "Filter by minimum tone"
// @Param tone_max query number false "Filter by maximum tone"
// @Success 200 {array} ds.Interval
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /intervals [get]
func (h *IntervalHandler) GetIntervals(ctx *gin.Context) {
	title := ctx.Query("title")
	toneMinStr := ctx.Query("tone_min")
	toneMaxStr := ctx.Query("tone_max")

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

	intervals, err := h.repo.Interval.GetIntervals(title, toneMin, toneMax)
	if err != nil {
		logrus.Error("Failed to get intervals: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get intervals"})
		return
	}

	ctx.JSON(http.StatusOK, intervals)
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
