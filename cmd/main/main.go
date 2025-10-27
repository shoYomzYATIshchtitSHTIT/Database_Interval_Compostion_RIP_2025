package main

import (
	"Backend-RIP/internal/app/config"
	"Backend-RIP/internal/app/repository"
	"Backend-RIP/internal/pkg"

	_ "Backend-RIP/docs" // Важно: добавляем импорт docs

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @title Composition Service API
// @version 1.0
// @description API for music composition service with JWT authentication and role-based access control

// @contact.name API Support
// @contact.url http://localhost:8080
// @contact.email support@composition-service.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer token. Example: "Bearer {token}"

// @tag.name Users
// @tag.description User management and authentication
// @tag.name Intervals
// @tag.description Music intervals management
// @tag.name Compositions
// @tag.description Composition requests management
// @tag.name CompositionIntervals
// @tag.description Management of intervals within compositions
func main() {
	router := gin.Default()

	// Загружаем конфигурацию
	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	// Инициализируем репозиторий
	repo, err := repository.NewRepository()
	if err != nil {
		logrus.Fatalf("error initializing repository: %v", err)
	}

	// Создаем приложение с конфигурацией
	application := pkg.NewApp(conf, router, repo)

	// Запускаем приложение
	application.RunApp()
}
