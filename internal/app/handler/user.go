package handler

import (
	"Backend-RIP/internal/app/config"
	"Backend-RIP/internal/app/ds"
	"Backend-RIP/internal/app/middleware"
	"Backend-RIP/internal/app/repository"
	"Backend-RIP/internal/app/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type UserHandler struct {
	repo *repository.Repository
}

func NewUserHandler(repo *repository.Repository) *UserHandler {
	return &UserHandler{
		repo: repo,
	}
}

type RegisterRequest struct {
	Login       string `json:"login" binding:"required"`
	Password    string `json:"password" binding:"required"`
	IsModerator bool   `json:"is_moderator"`
}

type LoginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateProfileRequest struct {
	Login    *string `json:"login"`
	Password *string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register godoc
// @Summary Register new user
// @Description Create a new user account
// @Tags Users
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User registration data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/register [post]
func (h *UserHandler) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	user := &ds.Users{
		Login:       req.Login,
		Password:    req.Password,
		IsModerator: req.IsModerator,
	}

	if err := h.repo.User.RegisterUser(user); err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user_id": user.User_ID,
	})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get authenticated user's profile information
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} ds.Users
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/profile [get]
func (h *UserHandler) GetProfile(ctx *gin.Context) {
	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	user, err := h.repo.User.GetUserProfile(userID)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update authenticated user's profile information
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body UpdateProfileRequest true "Profile update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/profile [put]
func (h *UserHandler) UpdateProfile(ctx *gin.Context) {
	var req UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	updates := make(map[string]interface{})
	if req.Login != nil {
		updates["login"] = *req.Login
	}
	if req.Password != nil {
		updates["password"] = *req.Password
	}

	if err := h.repo.User.UpdateUserProfile(userID, updates); err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT tokens
// @Tags Users
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/login [post]
func (h *UserHandler) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	user, err := h.repo.User.AuthenticateUser(req.Login, req.Password)
	if err != nil {
		logrus.Error(err)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Получаем конфигурацию для JWT
	cfg, err := config.NewConfig()
	if err != nil {
		logrus.Error("Failed to get config: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Генерируем access token
	accessToken, err := utils.GenerateAccessToken(user, cfg.JWTSecret, cfg.JWTAccessExpire)
	if err != nil {
		logrus.Error("Failed to generate access token: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Генерируем refresh token
	refreshToken, err := utils.GenerateRefreshToken(user, cfg.JWTSecret, cfg.JWTRefreshExpire)
	if err != nil {
		logrus.Error("Failed to generate refresh token: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Сохраняем refresh token в Redis (если доступен)
	if h.repo.GetRedisClient() != nil {
		err = h.repo.GetRedisClient().SaveRefreshToken(
			ctx.Request.Context(),
			user.User_ID,
			refreshToken,
			cfg.JWTRefreshExpire,
		)
		if err != nil {
			logrus.Error("Failed to save refresh token: ", err)
		}

		// СОХРАНЯЕМ СЕССИЮ ПОЛЬЗОВАТЕЛЯ
		sessionData := map[string]interface{}{
			"user_id":      user.User_ID,
			"login":        user.Login,
			"is_moderator": user.IsModerator,
			"login_time":   time.Now().Format(time.RFC3339),
			"ip_address":   ctx.ClientIP(),
		}

		err = h.repo.GetRedisClient().SaveUserSession(
			ctx.Request.Context(),
			user.User_ID,
			sessionData,
			cfg.JWTAccessExpire, // TTL такой же как у access token
		)
		if err != nil {
			logrus.Error("Failed to save user session: ", err)
		} else {
			logrus.Infof("User session saved for user_id: %d", user.User_ID)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_at":    time.Now().Add(cfg.JWTAccessExpire),
		"user_id":       user.User_ID,
		"login":         user.Login,
		"is_moderator":  user.IsModerator,
	})
}

// Logout godoc
// @Summary User logout
// @Description Invalidate user token
// @Tags Users
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/logout [post]
func (h *UserHandler) Logout(ctx *gin.Context) {
	// Получаем токен из заголовка
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header required"})
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Получаем конфигурацию
	cfg, err := config.NewConfig()
	if err != nil {
		logrus.Error("Failed to get config: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Добавляем токен в blacklist (если Redis доступен)
	if h.repo.GetRedisClient() != nil {
		err = h.repo.GetRedisClient().AddToBlacklist(
			ctx.Request.Context(),
			tokenString,
			cfg.JWTAccessExpire,
		)
		if err != nil {
			logrus.Error("Failed to add token to blacklist: ", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
			return
		}

		// УДАЛЯЕМ СЕССИЮ ПОЛЬЗОВАТЕЛЯ
		if userID, exists := middleware.GetUserID(ctx); exists {
			err = h.repo.GetRedisClient().DeleteUserSession(ctx.Request.Context(), userID)
			if err != nil {
				logrus.Error("Failed to delete user session: ", err)
			} else {
				logrus.Infof("User session deleted for user_id: %d", userID)
			}
		}
	}

	// Удаляем refresh token если пользователь аутентифицирован и Redis доступен
	if userID, exists := middleware.GetUserID(ctx); exists && h.repo.GetRedisClient() != nil {
		h.repo.GetRedisClient().DeleteRefreshToken(ctx.Request.Context(), userID)
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// RefreshToken обновляет access token
func (h *UserHandler) RefreshToken(ctx *gin.Context) {
	var req RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Получаем конфигурацию
	cfg, err := config.NewConfig()
	if err != nil {
		logrus.Error("Failed to get config: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Валидируем refresh token
	claims, err := utils.ValidateToken(req.RefreshToken, cfg.JWTSecret)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Проверяем, что refresh token есть в Redis
	if h.repo.GetRedisClient() != nil {
		storedToken, err := h.repo.GetRedisClient().GetRefreshToken(
			ctx.Request.Context(),
			claims.UserID,
		)
		if err != nil || storedToken != req.RefreshToken {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token not found"})
			return
		}
	}

	// Получаем пользователя из базы
	user, err := h.repo.User.GetUserByID(claims.UserID)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Генерируем новые токены
	accessToken, err := utils.GenerateAccessToken(user, cfg.JWTSecret, cfg.JWTAccessExpire)
	if err != nil {
		logrus.Error("Failed to generate access token: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user, cfg.JWTSecret, cfg.JWTRefreshExpire)
	if err != nil {
		logrus.Error("Failed to generate refresh token: ", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Обновляем refresh token в Redis
	if h.repo.GetRedisClient() != nil {
		err = h.repo.GetRedisClient().SaveRefreshToken(
			ctx.Request.Context(),
			user.User_ID,
			refreshToken,
			cfg.JWTRefreshExpire,
		)
		if err != nil {
			logrus.Error("Failed to save refresh token: ", err)
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_at":    time.Now().Add(cfg.JWTAccessExpire),
	})
}
