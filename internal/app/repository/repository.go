package repository

import (
	"Backend-RIP/internal/app/config"
	"Backend-RIP/internal/app/dsn"
	"Backend-RIP/internal/app/redis"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository struct {
	db                   *gorm.DB
	redisClient          *redis.Client
	Interval             *IntervalRepository
	Composition_interval *CompositionIntervalRepository
	User                 *UserRepository
}

func NewRepository() (*Repository, error) {
	// Загружаем конфигурацию
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, err
	}

	// Инициализируем базу данных
	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Инициализируем Redis клиент
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		logrus.Warnf("Failed to initialize Redis client: %v", err)
		// Продолжаем без Redis, но логируем предупреждение
	}

	// Инициализируем MinIO клиент
	minioClient, err := InitMinIOClient()
	if err != nil {
		return nil, err
	}

	// Создаем репозиторий
	repo := &Repository{
		db:                   db,
		redisClient:          redisClient,
		Interval:             NewIntervalRepository(db, minioClient),
		Composition_interval: NewCompositionIntervalRepository(db),
		User:                 NewUserRepository(db),
	}

	return repo, nil
}

// GetRedisClient возвращает Redis клиент
func (r *Repository) GetRedisClient() *redis.Client {
	return r.redisClient
}

// Close закрывает все соединения
func (r *Repository) Close() {
	if r.redisClient != nil {
		if err := r.redisClient.Close(); err != nil {
			logrus.Errorf("Error closing Redis client: %v", err)
		}
	}
}

// InitMinIOClient (существующий код без изменений)
func InitMinIOClient() (*minio.Client, error) {
	endpoint := "localhost:9000"
	accessKeyID := "minio"
	secretAccessKey := "minio124"
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %v", err)
	}

	ctx := context.Background()

	// Проверяем подключение
	_, err = minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("minio connection test failed: %v", err)
	}

	// Создаем bucket если не существует
	exists, err := minioClient.BucketExists(ctx, "interval-image")
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %v", err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, "interval-image", minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %v", err)
		}
	}

	logrus.Info("MinIO client initialized successfully")
	return minioClient, nil
}
