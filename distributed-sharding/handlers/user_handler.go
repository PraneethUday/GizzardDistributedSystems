package handlers

import (
	"net/http"
	"strconv"
	"sync"

	"distributed-sharding/models"
	"distributed-sharding/repository"
	"distributed-sharding/sharding"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests with sharding support
type UserHandler struct {
	shardManager *sharding.ShardManager
	idCounter    int
	mu           sync.Mutex
}

// NewUserHandler creates a new UserHandler with the given ShardManager
func NewUserHandler(sm *sharding.ShardManager) *UserHandler {
	// Get the max ID across all shards to continue incrementing
	maxID := 0
	for _, db := range sm.GetAllShards() {
		var id int
		err := db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM users").Scan(&id)
		if err == nil && id > maxID {
			maxID = id
		}
	}
	return &UserHandler{
		shardManager: sm,
		idCounter:    maxID,
	}
}

// CreateUser handles POST /users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.UserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Generate new ID atomically
	h.mu.Lock()
	h.idCounter++
	newID := h.idCounter
	h.mu.Unlock()

	// Get the shard for this user ID using formula: (userID - 1) % 4
	// user 1 → shard1.db, user 2 → shard2.db, user 3 → shard3.db, user 4 → shard4.db
	db := h.shardManager.GetShard(newID)
	shardIndex := h.shardManager.GetShardIndex(newID)

	// Create user struct
	user := models.User{
		ID:    newID,
		Name:  req.Name,
		Email: req.Email,
	}

	// Insert user into the appropriate shard using repository function
	if err := repository.InsertUser(db, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "User created successfully",
		"user":     user,
		"shard_id": shardIndex,
	})
}

// GetUser handles GET /users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Calculate shard using userID % 4 formula
	// user 1 → shard1.db, user 2 → shard2.db, user 3 → shard3.db, user 4 → shard4.db
	db := h.shardManager.GetShard(id)
	shardIndex := h.shardManager.GetShardIndex(id)

	// Fetch the user from the correct shard using repository function
	user, err := repository.GetUser(db, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch user",
			"details": err.Error(),
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"shard_id": shardIndex,
	})
}

// GetAllUsers handles GET /users - retrieves users from all shards
func (h *UserHandler) GetAllUsers(c *gin.Context) {
	var allUsers []models.UserWithShard

	for i, db := range h.shardManager.GetAllShards() {
		shardID := i + 1
		users, err := repository.GetAllUsers(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to fetch users",
				"details": err.Error(),
			})
			return
		}

		for _, user := range users {
			allUsers = append(allUsers, models.UserWithShard{
				User:    user,
				ShardID: shardID,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users": allUsers,
		"count": len(allUsers),
	})
}

// GetShardStats handles GET /shards - returns shard statistics
func (h *UserHandler) GetShardStats(c *gin.Context) {
	stats, err := h.shardManager.GetShardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get shard stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"shards":      stats,
		"total_shards": h.shardManager.GetNumShards(),
	})
}

// HealthCheck handles GET /health
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "distributed-sharding-api",
	})
}
