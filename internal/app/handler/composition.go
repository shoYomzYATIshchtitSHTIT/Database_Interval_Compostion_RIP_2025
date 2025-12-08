package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"Backend-RIP/internal/app/middleware"
	"Backend-RIP/internal/app/repository"

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
	Title     *string `json:"title"`
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
			"title":        comp.Title,
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
// @Security BearerAuth
// @Produce json
// @Param id path int true "Composition ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /compositions/{id} [get]
func (h *CompositionHandler) GetComposition(ctx *gin.Context) {
	// Получаем информацию о пользователе из контекста
	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

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

	// Проверяем права доступа
	isModerator := middleware.IsModerator(ctx)

	// Гость (не модератор) может смотреть только свои заявки
	if !isModerator && composition.CreatorID != userID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
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
		"title":        composition.Title,
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
				"description": ci.Interval.Description,
				"tone":        ci.Interval.Tone,
				"photo":       ci.Interval.Photo,
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
// PUT изменения полей заявки
func (h *CompositionHandler) UpdateCompositionFields(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid composition ID"})
		return
	}

	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Загружаем заявку
	composition, err := h.repo.Composition_interval.GetComposition(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Composition not found"})
		return
	}

	// Разрешаем редактирование только если статус = Черновик
	if composition.Status != "Черновик" {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Only draft compositions can be edited"})
		return
	}

	// Только создатель может редактировать
	if composition.CreatorID != userID {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req UpdateCompositionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updates := map[string]interface{}{}
	if req.Belonging != nil {
		updates["belonging"] = *req.Belonging
	}
	if req.Title != nil {
		updates["title"] = *req.Title
	}

	if err := h.repo.Composition_interval.UpdateCompositionFields(uint(id), updates); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update composition"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Composition updated"})
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
// @Summary Complete composition and start async calculation
// @Description Complete composition and automatically start async calculation in Django service (moderator only)
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

	// Завершаем заявку
	calculationData := make(map[string]interface{})
	err = h.repo.Composition_interval.CompleteComposition(uint(id), moderatorID, "Завершена", calculationData)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ЗАПУСКАЕМ АСИНХРОННЫЙ РАСЧЁТ В DJANGO-СЕРВИСЕ
	go func(compositionID uint) {
		logrus.Infof("Starting async calculation for composition %d via Django service", compositionID)

		// Подготовка запроса к Django-сервису
		payload := map[string]interface{}{
			"composition_id": compositionID,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			logrus.Errorf("Failed to marshal request for composition %d: %v", compositionID, err)
			return
		}

		// Вызов Django-сервиса
		resp, err := http.Post(
			"http://localhost:8001/calculate/", // URL Django-сервиса
			"application/json",
			bytes.NewBuffer(jsonData),
		)

		if err != nil {
			logrus.Errorf("Failed to call Django service for composition %d: %v", compositionID, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logrus.Errorf("Django service returned error for composition %d: HTTP %d", compositionID, resp.StatusCode)
			// Читаем тело ответа для отладки
			body, _ := io.ReadAll(resp.Body)
			logrus.Errorf("Response body: %s", string(body))
		} else {
			logrus.Infof("Django service accepted calculation request for composition %d", compositionID)
		}
	}(uint(id))

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Composition completed successfully. Async calculation started in Django service.",
	})
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

type CalculationResultRequest struct {
	CompositionID uint   `json:"composition_id" binding:"required"`
	Result        string `json:"result" binding:"required"`
	APIKey        string `json:"api_key" binding:"required"`
}

// ReceiveCalculationResult godoc
// @Summary Получить результат расчёта от асинхронного сервиса
// @Description Принимает результат расчёта от Django-сервиса
// @Tags Compositions
// @Accept json
// @Produce json
// @Param request body CalculationResultRequest true "Результат расчёта"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /compositions/receive-result [post]
func (h *CompositionHandler) ReceiveCalculationResult(ctx *gin.Context) {
	var req CalculationResultRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные запроса: " + err.Error()})
		return
	}

	// Проверка API ключа (псевдоавторизация)
	const expectedAPIKey = "SECRET123" // Константа на 8 байт
	if req.APIKey != expectedAPIKey {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный API ключ"})
		return
	}

	// Валидация результата
	if req.Result != "принадлежит" && req.Result != "не принадлежит" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверное значение результата"})
		return
	}

	// Проверяем существование композиции (без сохранения в переменную)
	if _, err := h.repo.Composition_interval.GetComposition(req.CompositionID); err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Композиция не найдена"})
		return
	}

	// Обновляем поле belonging
	updates := map[string]interface{}{
		"belonging":   req.Result,
		"date_update": time.Now(),
	}

	err := h.repo.Composition_interval.UpdateCompositionFields(req.CompositionID, updates)
	if err != nil {
		logrus.Errorf("Failed to update calculation result for composition %d: %v", req.CompositionID, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обновлении результата"})
		return
	}

	logrus.Infof("Calculation result received and saved: composition %d -> %s", req.CompositionID, req.Result)
	ctx.JSON(http.StatusOK, gin.H{
		"message":        "Результат расчёта успешно обновлён",
		"composition_id": req.CompositionID,
		"result":         req.Result,
	})
}
