package middleware

import (
	"github.com/gin-gonic/gin"
)

// GetUserID возвращает ID пользователя из контекста
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetLogin возвращает логин пользователя из контекста
func GetLogin(c *gin.Context) (string, bool) {
	login, exists := c.Get("login")
	if !exists {
		return "", false
	}
	return login.(string), true
}

// IsModerator проверяет, является ли пользователь модератором
func IsModerator(c *gin.Context) bool {
	isModerator, exists := c.Get("is_moderator")
	if !exists {
		return false
	}
	return isModerator.(bool)
}

// RequireUserID middleware требует аутентификации и возвращает ошибку если пользователь не аутентифицирован
func RequireUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.JSON(401, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}
		c.Set("required_user_id", userID)
		c.Next()
	}
}
