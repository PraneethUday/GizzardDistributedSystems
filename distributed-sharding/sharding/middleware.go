package sharding

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ShardMiddleware provides middleware functions for shard-aware request handling
type ShardMiddleware struct {
	manager *ShardManager
}

// NewShardMiddleware creates a new ShardMiddleware
func NewShardMiddleware(manager *ShardManager) *ShardMiddleware {
	return &ShardMiddleware{manager: manager}
}

// ExtractAndSetShard extracts userID from the request and sets the appropriate shard
// The shard connection is stored in the Gin context for use by handlers
func (sm *ShardMiddleware) ExtractAndSetShard() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to extract userID from URL parameter
		userIDStr := c.Param("id")
		if userIDStr != "" {
			userID, err := strconv.Atoi(userIDStr)
			if err == nil {
				shard := sm.manager.GetShard(userID)
				shardIndex := sm.manager.GetShardIndex(userID)
				c.Set("db", shard)
				c.Set("shardIndex", shardIndex)
				c.Set("userID", userID)
			}
		}
		c.Next()
	}
}

// GetShardFromContext retrieves the database shard from the Gin context
func GetShardFromContext(c *gin.Context) (*sql.DB, bool) {
	db, exists := c.Get("db")
	if !exists {
		return nil, false
	}
	return db.(*sql.DB), true
}

// GetShardIndexFromContext retrieves the shard index from the Gin context
func GetShardIndexFromContext(c *gin.Context) (int, bool) {
	index, exists := c.Get("shardIndex")
	if !exists {
		return 0, false
	}
	return index.(int), true
}

// RequireShard middleware ensures a valid shard is available
func (sm *ShardMiddleware) RequireShard() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := GetShardFromContext(c)
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unable to determine database shard",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetManager returns the underlying ShardManager
func (sm *ShardMiddleware) GetManager() *ShardManager {
	return sm.manager
}
