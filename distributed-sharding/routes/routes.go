package routes

import (
	"distributed-sharding/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine, userHandler *handlers.UserHandler) {
	// Health check endpoint
	router.GET("/health", handlers.HealthCheck)

	// User endpoints
	router.POST("/users", userHandler.CreateUser)
	router.GET("/users", userHandler.GetAllUsers)
	router.GET("/users/:id", userHandler.GetUser)

	// Shard management endpoints
	router.GET("/shards", userHandler.GetShardStats)
}
