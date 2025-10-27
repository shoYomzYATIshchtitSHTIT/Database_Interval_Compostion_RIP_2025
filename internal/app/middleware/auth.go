package middleware

import (
	"Backend-RIP/internal/app/config"
	"Backend-RIP/internal/app/ds"
	"Backend-RIP/internal/app/repository"
	"Backend-RIP/internal/app/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	jwtPrefix = "Bearer "
)

// AuthMiddleware проверяет JWT токен и добавляет пользователя в контекст
func AuthMiddleware(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем заголовок Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Проверяем формат Bearer токена
		if !strings.HasPrefix(authHeader, jwtPrefix) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		// Извлекаем токен
		tokenString := strings.TrimPrefix(authHeader, jwtPrefix)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is empty"})
			c.Abort()
			return
		}

		// Получаем конфигурацию
		cfg, err := getConfig()
		if err != nil {
			logrus.Error("Failed to get config: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Проверяем токен в blacklist (если Redis доступен)
		if repo.GetRedisClient() != nil {
			inBlacklist, err := repo.GetRedisClient().IsInBlacklist(c.Request.Context(), tokenString)
			if err != nil {
				logrus.Error("Failed to check token in blacklist: ", err)
			} else if inBlacklist {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is invalidated"})
				c.Abort()
				return
			}
		}

		// Валидируем токен
		claims, err := utils.ValidateToken(tokenString, cfg.JWTSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Добавляем информацию о пользователе в контекст
		c.Set("user_id", claims.UserID)
		c.Set("login", claims.Login)
		c.Set("is_moderator", claims.IsModerator)

		logrus.Debugf("User authenticated: %s (ID: %d, Moderator: %t)",
			claims.Login, claims.UserID, claims.IsModerator)

		c.Next()
	}
}

// ModeratorOnly middleware проверяет, что пользователь является модератором
func ModeratorOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		isModerator, exists := c.Get("is_moderator")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		if !isModerator.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Moderator access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth middleware добавляет пользователя в контекст если токен валиден, но не требует аутентификации
func OptionalAuth(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, jwtPrefix) {
			c.Next()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, jwtPrefix)
		if tokenString == "" {
			c.Next()
			return
		}

		cfg, err := getConfig()
		if err != nil {
			logrus.Error("Failed to get config: ", err)
			c.Next()
			return
		}

		// Проверяем blacklist
		if repo.GetRedisClient() != nil {
			inBlacklist, err := repo.GetRedisClient().IsInBlacklist(c.Request.Context(), tokenString)
			if err == nil && inBlacklist {
				c.Next()
				return
			}
		}

		// Валидируем токен
		claims, err := utils.ValidateToken(tokenString, cfg.JWTSecret)
		if err != nil {
			c.Next()
			return
		}

		// Добавляем информацию о пользователе в контекст
		c.Set("user_id", claims.UserID)
		c.Set("login", claims.Login)
		c.Set("is_moderator", claims.IsModerator)

		c.Next()
	}
}

// GetUserFromContext извлекает информацию о пользователе из контекста
func GetUserFromContext(c *gin.Context) (*ds.JWTClaims, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, false
	}

	login, exists := c.Get("login")
	if !exists {
		return nil, false
	}

	isModerator, exists := c.Get("is_moderator")
	if !exists {
		return nil, false
	}

	return &ds.JWTClaims{
		UserID:      userID.(uint),
		Login:       login.(string),
		IsModerator: isModerator.(bool),
	}, true
}

// getConfig вспомогательная функция для получения конфигурации
func getConfig() (*config.Config, error) {
	return config.NewConfig()
}
