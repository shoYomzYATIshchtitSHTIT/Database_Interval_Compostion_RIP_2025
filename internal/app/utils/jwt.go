package utils

import (
	"Backend-RIP/internal/app/ds"
	"time"

	"github.com/golang-jwt/jwt"
)

func GenerateAccessToken(user ds.Users, secret string, expiresIn time.Duration) (string, error) {
	claims := ds.JWTClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(expiresIn).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "composition-service",
			Subject:   user.Login,
		},
		UserID:      user.User_ID,
		Login:       user.Login,
		IsModerator: user.IsModerator,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func GenerateRefreshToken(user ds.Users, secret string, expiresIn time.Duration) (string, error) {
	claims := ds.JWTClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(expiresIn).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "composition-service",
			Subject:   user.Login,
		},
		UserID:      user.User_ID,
		Login:       user.Login,
		IsModerator: user.IsModerator,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString string, secret string) (*ds.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ds.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*ds.JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}
