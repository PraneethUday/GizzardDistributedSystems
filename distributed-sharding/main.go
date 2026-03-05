package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"distributed-sharding/handlers"
	"distributed-sharding/routes"
	"distributed-sharding/sharding"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create data directory for shard databases
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize ShardManager with 4 shards
	shardManager, err := sharding.NewShardManager(dataDir, 4)
	if err != nil {
		log.Fatalf("Failed to initialize shard manager: %v", err)
	}

	// Ensure shards are closed on shutdown
	defer shardManager.Close()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		shardManager.Close()
		os.Exit(0)
	}()

	// Create user handler with shard manager
	userHandler := handlers.NewUserHandler(shardManager)

	// Create Gin router with default middleware (logger and recovery)
	router := gin.Default()

	// Enable CORS for all origins (useful for React frontend)
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Setup routes with user handler
	routes.SetupRoutes(router, userHandler)

	// Start server
	log.Println("Starting Distributed Sharding API server on :8080")
	log.Println("Shards initialized: shard1.db, shard2.db, shard3.db, shard4.db")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
