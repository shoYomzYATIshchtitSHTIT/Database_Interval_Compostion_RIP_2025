package ds

import (
	"time"

	"github.com/golang-jwt/jwt"
)

type JWTClaims struct {
	jwt.StandardClaims
	UserID      uint   `json:"user_id"`
	Login       string `json:"login"`
	IsModerator bool   `json:"is_moderator"`
}

type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       uint      `json:"user_id"`
	Login        string    `json:"login"`
	IsModerator  bool      `json:"is_moderator"`
}

type LoginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}
