package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserRequest represents the request body for creating a user
type UserRequest struct {
	ID    int    `json:"id" binding:"required"`
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

// ShardNode represents a single shard node server
type ShardNode struct {
	db      *sql.DB
	shardID int
	port    int
}

// NewShardNode creates a new shard node
func NewShardNode(shardID, port int, dataDir string) (*ShardNode, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open SQLite database for this shard
	dbPath := fmt.Sprintf("%s/shard%d.db", dataDir, shardID)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize schema
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		name TEXT,
		email TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Printf("Shard %d initialized with database at %s", shardID, dbPath)

	return &ShardNode{
		db:      db,
		shardID: shardID,
		port:    port,
	}, nil
}

// InsertUser inserts a user into this shard
func (n *ShardNode) InsertUser(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Insert user into database
	_, err := n.db.Exec("INSERT INTO users (id, name, email) VALUES (?, ?, ?)",
		req.ID, req.Name, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to insert user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "User created successfully",
		"user":     User{ID: req.ID, Name: req.Name, Email: req.Email},
		"shard_id": n.shardID,
	})
}

// GetUser retrieves a user from this shard
func (n *ShardNode) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var user User
	err = n.db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id).
		Scan(&user.ID, &user.Name, &user.Email)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"shard_id": n.shardID,
	})
}

// GetAllUsers retrieves all users from this shard
func (n *ShardNode) GetAllUsers(c *gin.Context) {
	rows, err := n.db.Query("SELECT id, name, email FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch users",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			continue
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"users":    users,
		"count":    len(users),
		"shard_id": n.shardID,
	})
}

// HealthCheck returns the health status of this shard
func (n *ShardNode) HealthCheck(c *gin.Context) {
	err := n.db.Ping()
	status := "healthy"
	if err != nil {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   status,
		"shard_id": n.shardID,
		"port":     n.port,
	})
}

// Close closes the database connection
func (n *ShardNode) Close() error {
	return n.db.Close()
}

func main() {
	shardID := flag.Int("shard", 1, "Shard ID (1-4)")
	port := flag.Int("port", 8001, "Port to run the server on")
	dataDir := flag.String("data", "./data", "Data directory for SQLite databases")
	flag.Parse()

	if *shardID < 1 || *shardID > 4 {
		log.Fatalf("Invalid shard ID: %d. Must be between 1 and 4", *shardID)
	}

	node, err := NewShardNode(*shardID, *port, *dataDir)
	if err != nil {
		log.Fatalf("Failed to create shard node: %v", err)
	}
	defer node.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Printf("Shutting down shard %d...", *shardID)
		node.Close()
		os.Exit(0)
	}()

	router := gin.Default()

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

	router.GET("/health", node.HealthCheck)
	router.POST("/insert", node.InsertUser)
	router.GET("/user/:id", node.GetUser)
	router.GET("/users", node.GetAllUsers)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting Shard %d server on port %d", *shardID, *port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
