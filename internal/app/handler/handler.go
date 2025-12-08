package handler

import (
	"Backend-RIP/internal/app/middleware"
	"Backend-RIP/internal/app/repository"

	"github.com/gin-gonic/gin"
)

func RegisterHandlers(router *gin.Engine, repo *repository.Repository) {
	apiRouter := router.Group("/api")

	intervalHandler := NewIntervalHandler(repo)
	compositionHandler := NewCompositionHandler(repo)
	compositionIntervalHandler := NewCompositionIntervalHandler(repo)
	userHandler := NewUserHandler(repo)

	// Public routes - доступны без аутентификации
	public := apiRouter.Group("")
	{
		// Аутентификация
		public.POST("/users/login", userHandler.Login)
		public.POST("/users/register", userHandler.Register)
		public.POST("/users/refresh", userHandler.RefreshToken)

		// Просмотр интервалов (доступно всем)
		public.GET("/intervals", intervalHandler.GetIntervals)
		public.GET("/intervals/:id", intervalHandler.GetInterval)

		public.POST("/compositions/receive-result", compositionHandler.ReceiveCalculationResult)
	}

	// Protected routes - требуют аутентификации
	protected := apiRouter.Group("")
	protected.Use(middleware.AuthMiddleware(repo))
	{
		protected.GET("/users/profile", userHandler.GetProfile)
		protected.PUT("/users/profile", userHandler.UpdateProfile)
		protected.POST("/users/logout", userHandler.Logout)

		protected.GET("/compositions", compositionHandler.GetCompositions)
		protected.GET("/compositions/comp-cart", compositionHandler.GetCompositionCart)
		protected.GET("/compositions/:id", compositionHandler.GetComposition)
		protected.POST("/intervals/add-to-composition", intervalHandler.AddIntervalToComposition)
		protected.PUT("/compositions/:id/form", compositionHandler.FormComposition)
		protected.DELETE("/compositions/:id", compositionHandler.DeleteComposition)

		protected.PUT("/compositions/:id", compositionHandler.UpdateCompositionFields)

		protected.DELETE("/composition-intervals", compositionIntervalHandler.RemoveFromComposition)
		protected.PUT("/composition-intervals", compositionIntervalHandler.UpdateCompositionInterval)
	}

	moderator := apiRouter.Group("")
	moderator.Use(middleware.AuthMiddleware(repo), middleware.ModeratorOnly())
	{
		moderator.POST("/intervals", intervalHandler.CreateInterval)
		moderator.PUT("/intervals/:id", intervalHandler.UpdateInterval)
		moderator.DELETE("/intervals/:id", intervalHandler.DeleteInterval)
		moderator.POST("/intervals/:id/image", intervalHandler.UpdateIntervalPhoto)

		moderator.PUT("/compositions/:id/complete", compositionHandler.CompleteComposition)
		moderator.PUT("/compositions/:id/reject", compositionHandler.RejectComposition)

	}
}
