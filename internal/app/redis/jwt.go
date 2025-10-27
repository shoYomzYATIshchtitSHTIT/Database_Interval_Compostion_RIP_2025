package redis

import (
	"context"
	"fmt"
	"time"
)

const (
	// Префиксы для ключей Redis
	blacklistPrefix    = "jwt:blacklist:"
	userSessionPrefix  = "user:session:"
	refreshTokenPrefix = "refresh:token:"
)

// AddToBlacklist добавляет JWT токен в черный список
func (c *Client) AddToBlacklist(ctx context.Context, token string, expiresIn time.Duration) error {
	key := blacklistPrefix + token
	return c.Set(ctx, key, "blacklisted", expiresIn)
}

// IsInBlacklist проверяет, находится ли токен в черном списке
func (c *Client) IsInBlacklist(ctx context.Context, token string) (bool, error) {
	key := blacklistPrefix + token
	exists, err := c.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %v", err)
	}
	return exists, nil
}

// SaveRefreshToken сохраняет refresh token для пользователя
func (c *Client) SaveRefreshToken(ctx context.Context, userID uint, refreshToken string, expiresIn time.Duration) error {
	key := refreshTokenPrefix + fmt.Sprintf("%d", userID)
	return c.Set(ctx, key, refreshToken, expiresIn)
}

// GetRefreshToken получает refresh token пользователя
func (c *Client) GetRefreshToken(ctx context.Context, userID uint) (string, error) {
	key := refreshTokenPrefix + fmt.Sprintf("%d", userID)
	return c.Get(ctx, key)
}

// DeleteRefreshToken удаляет refresh token пользователя
func (c *Client) DeleteRefreshToken(ctx context.Context, userID uint) error {
	key := refreshTokenPrefix + fmt.Sprintf("%d", userID)
	return c.Delete(ctx, key)
}

// SaveUserSession сохраняет информацию о сессии пользователя
func (c *Client) SaveUserSession(ctx context.Context, userID uint, sessionData map[string]interface{}, expiresIn time.Duration) error {
	key := userSessionPrefix + fmt.Sprintf("%d", userID)

	// Конвертируем map в Redis HSet
	for field, value := range sessionData {
		err := c.client.HSet(ctx, key, field, value).Err()
		if err != nil {
			return err
		}
	}

	// Устанавливаем TTL для всей хэш-таблицы
	return c.Expire(ctx, key, expiresIn)
}

// GetUserSession получает информацию о сессии пользователя
func (c *Client) GetUserSession(ctx context.Context, userID uint) (map[string]string, error) {
	key := userSessionPrefix + fmt.Sprintf("%d", userID)
	return c.client.HGetAll(ctx, key).Result()
}

// DeleteUserSession удаляет сессию пользователя
func (c *Client) DeleteUserSession(ctx context.Context, userID uint) error {
	key := userSessionPrefix + fmt.Sprintf("%d", userID)
	return c.Delete(ctx, key)
}
