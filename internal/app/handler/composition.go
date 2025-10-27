package handler

import (
	"Backend-RIP/internal/app/middleware"
	"Backend-RIP/internal/app/repository"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CompositionHandler struct {
	repo *repository.Repository
}

func NewCompositionHandler(repo *repository.Repository) *CompositionHandler {
	return &CompositionHandler{
		repo: repo,
	}
}

type CartInfoResponse struct {
	CompositionID uint  `json:"composition_id"`
	ItemCount     int64 `json:"item_count"`
}

type UpdateCompositionRequest struct {
	Belonging *string `json:"belonging"`
}

// GetCompositionCart godoc
// @Summary Get composition cart
// @Description Get user's draft composition with item count
// @Tags Compositions
// @Security BearerAuth
// @Produce json
// @Success 200 {object} CartInfoResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /compositions/comp-cart [get]
func (h *CompositionHandler) GetCompositionCart(ctx *gin.Context) {
	creatorID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	compositionID, itemCount, err := h.repo.Composition_interval.GetCompositionCart(creatorID)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get composition cart"})
		return
	}

	ctx.JSON(http.StatusOK, CartInfoResponse{
		CompositionID: compositionID,
		ItemCount:     itemCount,
	})
}

// GetCompositions godoc
// @Summary Get compositions list
// @Description Get list of compositions with filtering (authenticated users only)
// @Tags Compositions
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status"
// @Param date_from query string false "Filter by date from (YYYY-MM-DD)"
// @Param date_to query string false "Filter by date to (YYYY-MM-DD)"
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /compositions [get]
func (h *CompositionHandler) GetCompositions(ctx *gin.Context) {
	status := ctx.Query("status")

	var dateFrom, dateTo time.Time
	if dateFromStr := ctx.Query("date_from"); dateFromStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			dateFrom = parsed
		}
	}
	if dateToStr := ctx.Query("date_to"); dateToStr != "" {
		if parsed, err := time.Parse("2006-01-02", dateToStr); err == nil {
			dateTo = parsed
		}
	}

	// Получаем информацию о пользователе из контекста
	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	isModerator := middleware.IsModerator(ctx)

	compositions, err := h.repo.Composition_interval.GetCompositions(status, dateFrom, dateTo, userID, isModerator)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get compositions"})
		return
	}

	response := make([]map[string]interface{}, 0)
	for _, comp := range compositions {
		item := map[string]interface{}{
			"id":           comp.ID,
			"status":       comp.Status,
			"creator_id":   comp.CreatorID,
			"moderator_id": comp.ModeratorID,
			"date_create":  comp.DateCreate.Format("2006-01-02 15:04:05"),
			"date_update":  comp.DateUpdate.Format("2006-01-02 15:04:05"),
			"belonging":    comp.Belonging,
		}

		if comp.DateFinish.Valid {
			item["date_finish"] = comp.DateFinish.Time.Format("2006-01-02 15:04:05")
		}

		response = append(response, item)
	}

	ctx.JSON(http.StatusOK, response)
}

// GetComposition godoc
// @Summary Get composition details
// @Description Get composition details with intervals
// @Tags Compositions
// @Produce json
// @Param id path int true "Composition ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /compositions/{id} [get]
func (h *CompositionHandler) GetComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	composition, err := h.repo.Composition_interval.GetComposition(uint(id))
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Composition not found"})
		return
	}

	response := map[string]interface{}{
		"id":           composition.ID,
		"status":       composition.Status,
		"creator_id":   composition.CreatorID,
		"moderator_id": composition.ModeratorID,
		"date_create":  composition.DateCreate.Format("2006-01-02 15:04:05"),
		"date_update":  composition.DateUpdate.Format("2006-01-02 15:04:05"),
		"belonging":    composition.Belonging,
		"intervals":    []map[string]interface{}{},
	}

	if composition.DateFinish.Valid {
		response["date_finish"] = composition.DateFinish.Time.Format("2006-01-02 15:04:05")
	}

	if composition.CompositorIntervals != nil {
		intervals := make([]map[string]interface{}, 0)
		for _, ci := range composition.CompositorIntervals {
			intervalItem := map[string]interface{}{
				"interval_id": ci.IntervalID,
				"title":       ci.Interval.Title,
				"amount":      ci.Amount,
			}
			intervals = append(intervals, intervalItem)
		}
		response["intervals"] = intervals
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateCompositionFields godoc
// @Summary Update composition fields
// @Description Update composition fields
// @Tags Compositions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Composition ID"
// @Param request body UpdateCompositionRequest true "Update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /compositions/{id} [put]
func (h *CompositionHandler) UpdateCompositionFields(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	var req UpdateCompositionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	updates := make(map[string]interface{})
	if req.Belonging != nil {
		updates["belonging"] = *req.Belonging
	}

	err = h.repo.Composition_interval.UpdateCompositionFields(uint(id), updates)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update composition"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition updated successfully"})
}

// FormComposition godoc
// @Summary Form composition
// @Description Form composition from draft status (creator only)
// @Tags Compositions
// @Security BearerAuth
// @Param id path int true "Composition ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /compositions/{id}/form [put]
func (h *CompositionHandler) FormComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	creatorID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	err = h.repo.Composition_interval.FormComposition(uint(id), creatorID)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition formed successfully"})
}

// CompleteComposition godoc
// @Summary Complete composition
// @Description Complete composition (moderator only)
// @Tags Compositions
// @Security BearerAuth
// @Param id path int true "Composition ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /compositions/{id}/complete [put]
func (h *CompositionHandler) CompleteComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	moderatorID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	calculationData := make(map[string]interface{})
	err = h.repo.Composition_interval.CompleteComposition(uint(id), moderatorID, "Завершена", calculationData)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition completed successfully"})
}

// RejectComposition godoc
// @Summary Reject composition
// @Description Reject composition (moderator only)
// @Tags Compositions
// @Security BearerAuth
// @Param id path int true "Composition ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /compositions/{id}/reject [put]
func (h *CompositionHandler) RejectComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	moderatorID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	calculationData := make(map[string]interface{})
	err = h.repo.Composition_interval.CompleteComposition(uint(id), moderatorID, "Отклонена", calculationData)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition rejected successfully"})
}

// DeleteComposition godoc
// @Summary Delete composition
// @Description Delete composition (creator only)
// @Tags Compositions
// @Security BearerAuth
// @Param id path int true "Composition ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /compositions/{id} [delete]
func (h *CompositionHandler) DeleteComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	_, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	err = h.repo.Composition_interval.DeleteComposition(uint(id))
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete composition"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition deleted successfully"})
}
