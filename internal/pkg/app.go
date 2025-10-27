package pkg

import (
	"fmt"

	"Backend-RIP/internal/app/config"
	"Backend-RIP/internal/app/handler"
	"Backend-RIP/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Application struct {
	Config *config.Config
	Router *gin.Engine
	Repo   *repository.Repository
}

func NewApp(c *config.Config, r *gin.Engine, repo *repository.Repository) *Application {
	return &Application{
		Config: c,
		Router: r,
		Repo:   repo,
	}
}

func (a *Application) RunApp() {
	logrus.Info("Server start up")

	// Swagger документация
	a.Router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Регистрируем обработчики
	handler.RegisterHandlers(a.Router, a.Repo)

	// Статические файлы (если нужны)
	a.Router.LoadHTMLGlob("templates/*")
	a.Router.Static("/styles", "./resources/styles")
	a.Router.Static("/img", "./resources/img")

	serverAddress := fmt.Sprintf("%s:%d", a.Config.ServiceHost, a.Config.ServicePort)
	logrus.Infof("Starting server on %s", serverAddress)

	if err := a.Router.Run(serverAddress); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Server down")
}
