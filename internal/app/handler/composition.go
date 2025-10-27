package handler

import (
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

// GET иконки корзины
func (h *CompositionHandler) GetCompositionCart(ctx *gin.Context) {
	creatorID := uint(1) // Фиксированный пользователь-создатель
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

// GET список заявок с фильтрацией
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

	compositions, err := h.repo.Composition_interval.GetCompositions(status, dateFrom, dateTo)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get compositions"})
		return
	}

	response := make([]gin.H, 0)
	for _, comp := range compositions {
		item := gin.H{
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

// GET одна запись заявки
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

	response := gin.H{
		"id":           composition.ID,
		"status":       composition.Status,
		"creator_id":   composition.CreatorID,
		"moderator_id": composition.ModeratorID,
		"date_create":  composition.DateCreate.Format("2006-01-02 15:04:05"),
		"date_update":  composition.DateUpdate.Format("2006-01-02 15:04:05"),
		"belonging":    composition.Belonging,
		"intervals":    []gin.H{},
	}

	if composition.DateFinish.Valid {
		response["date_finish"] = composition.DateFinish.Time.Format("2006-01-02 15:04:05")
	}

	if composition.CompositorIntervals != nil {
		intervals := make([]gin.H, 0)
		for _, ci := range composition.CompositorIntervals {
			intervalItem := gin.H{
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

// PUT изменения полей заявки
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

// PUT сформировать создателем
func (h *CompositionHandler) FormComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	creatorID := uint(1) // Фиксированный пользователь-создатель
	err = h.repo.Composition_interval.FormComposition(uint(id), creatorID)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition formed successfully"})
}

// PUT завершить модератором
func (h *CompositionHandler) CompleteComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	moderatorID := uint(2)                          // Фиксированный модератор
	calculationData := make(map[string]interface{}) // Дополнительные расчетные данные если нужны

	err = h.repo.Composition_interval.CompleteComposition(uint(id), moderatorID, "Завершена", calculationData)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition completed successfully"})
}

// PUT отклонить модератором
func (h *CompositionHandler) RejectComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	moderatorID := uint(2) // Фиксированный модератор
	calculationData := make(map[string]interface{})

	err = h.repo.Composition_interval.CompleteComposition(uint(id), moderatorID, "Отклонена", calculationData)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition rejected successfully"})
}

// DELETE удаление заявки
func (h *CompositionHandler) DeleteComposition(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
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

// DELETE удаление интервала из заявки
func (h *CompositionHandler) DeleteCompositionInterval(ctx *gin.Context) {
	compositionIDStr := ctx.Param("composition_id")
	compositionID, err := strconv.ParseUint(compositionIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	intervalIDStr := ctx.Param("interval_id")
	intervalID, err := strconv.ParseUint(intervalIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval ID"})
		return
	}

	err = h.repo.Composition_interval.DeleteCompositionInterval(uint(compositionID), uint(intervalID))
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete interval from composition"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Interval removed from composition successfully"})
}

// PUT изменение количества интервалов в заявке
func (h *CompositionHandler) UpdateCompositionInterval(ctx *gin.Context) {
	compositionIDStr := ctx.Param("composition_id")
	compositionID, err := strconv.ParseUint(compositionIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	intervalIDStr := ctx.Param("interval_id")
	intervalID, err := strconv.ParseUint(intervalIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval ID"})
		return
	}

	var request struct {
		Amount uint `json:"amount" binding:"required,min=1"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	err = h.repo.Composition_interval.UpdateCompositionInterval(uint(compositionID), uint(intervalID), request.Amount)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update interval amount"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Interval amount updated successfully"})
}
