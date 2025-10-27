package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ServiceHost string
	ServicePort int

	// JWT Configuration
	JWTSecret        string
	JWTAccessExpire  time.Duration
	JWTRefreshExpire time.Duration

	// Redis Configuration
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
}

func NewConfig() (*Config, error) {
	var err error

	// Загружаем .env файл
	_ = godotenv.Load()

	// Загружаем TOML конфигурацию
	configName := "config"
	if os.Getenv("CONFIG_NAME") != "" {
		configName = os.Getenv("CONFIG_NAME")
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.WatchConfig()

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}

	// Загружаем JWT конфигурацию из .env
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-default-secret-key-for-development-change-in-production"
		log.Warn("Using default JWT secret - change in production!")
	}
	cfg.JWTSecret = jwtSecret

	// JWT Expire times
	accessExpire := 24 * time.Hour
	if exp := os.Getenv("JWT_ACCESS_EXPIRE"); exp != "" {
		if parsed, err := time.ParseDuration(exp); err == nil {
			accessExpire = parsed
		}
	}
	cfg.JWTAccessExpire = accessExpire

	refreshExpire := 168 * time.Hour
	if exp := os.Getenv("JWT_REFRESH_EXPIRE"); exp != "" {
		if parsed, err := time.ParseDuration(exp); err == nil {
			refreshExpire = parsed
		}
	}
	cfg.JWTRefreshExpire = refreshExpire

	// Redis конфигурация из .env
	cfg.RedisHost = getEnv("REDIS_HOST", "localhost")
	cfg.RedisPort = getEnv("REDIS_PORT", "6379")
	cfg.RedisPassword = getEnv("REDIS_PASSWORD", "")

	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			redisDB = db
		}
	}
	cfg.RedisDB = redisDB

	log.Info("config parsed")

	return cfg, nil
}

// getEnv вспомогательная функция для получения environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
