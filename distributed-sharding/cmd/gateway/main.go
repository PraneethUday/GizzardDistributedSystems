package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ShardConfig holds the configuration for a shard node
type ShardConfig struct {
	ShardID int
	Host    string
	Port    int
}

// Gateway is the central API gateway that routes requests to shard nodes
type Gateway struct {
	shards     []ShardConfig
	httpClient *http.Client
	numShards  int
}

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// InsertRequest represents the request body for creating a user
type InsertRequest struct {
	ID    int    `json:"id"`
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required"`
}

// NewGateway creates a new gateway with the given shard configurations
func NewGateway(shards []ShardConfig) *Gateway {
	return &Gateway{
		shards: shards,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		numShards: len(shards),
	}
}

// GetShardForUser returns the shard configuration for a given user ID
func (g *Gateway) GetShardForUser(userID int) ShardConfig {
	shardIndex := (userID - 1) % g.numShards
	if shardIndex < 0 {
		shardIndex = 0
	}
	return g.shards[shardIndex]
}

// forwardRequest forwards an HTTP request to the appropriate shard
func (g *Gateway) forwardRequest(method, url string, body []byte) ([]byte, int, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to forward request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// InsertUser handles POST /users
func (g *Gateway) InsertUser(c *gin.Context) {
	var req InsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	shard := g.GetShardForUser(req.ID)
	shardURL := fmt.Sprintf("http://%s:%d/insert", shard.Host, shard.Port)

	log.Printf("Routing user %d to shard %d at %s:%d", req.ID, shard.ShardID, shard.Host, shard.Port)

	bodyBytes, _ := json.Marshal(req)
	respBody, statusCode, err := g.forwardRequest("POST", shardURL, bodyBytes)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "Failed to reach shard node",
			"details": err.Error(),
			"shard":   shard.ShardID,
		})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// GetUser handles GET /users/:id
func (g *Gateway) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	shard := g.GetShardForUser(id)
	shardURL := fmt.Sprintf("http://%s:%d/user/%d", shard.Host, shard.Port, id)

	log.Printf("Routing GET user %d to shard %d at %s:%d", id, shard.ShardID, shard.Host, shard.Port)

	respBody, statusCode, err := g.forwardRequest("GET", shardURL, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "Failed to reach shard node",
			"details": err.Error(),
			"shard":   shard.ShardID,
		})
		return
	}

	c.Data(statusCode, "application/json", respBody)
}

// GetAllUsers handles GET /users
func (g *Gateway) GetAllUsers(c *gin.Context) {
	var allUsers []map[string]interface{}
	var errors []string

	for _, shard := range g.shards {
		shardURL := fmt.Sprintf("http://%s:%d/users", shard.Host, shard.Port)
		respBody, statusCode, err := g.forwardRequest("GET", shardURL, nil)
		if err != nil {
			errors = append(errors, fmt.Sprintf("shard %d: %v", shard.ShardID, err))
			continue
		}

		if statusCode == http.StatusOK {
			var shardResp map[string]interface{}
			if err := json.Unmarshal(respBody, &shardResp); err == nil {
				if users, ok := shardResp["users"].([]interface{}); ok {
					for _, u := range users {
						if userMap, ok := u.(map[string]interface{}); ok {
							userMap["shard_id"] = shard.ShardID
							allUsers = append(allUsers, userMap)
						}
					}
				}
			}
		}
	}

	response := gin.H{
		"users": allUsers,
		"count": len(allUsers),
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}

// GetShardStatus handles GET /shards
func (g *Gateway) GetShardStatus(c *gin.Context) {
	type ShardStatus struct {
		ShardID   int    `json:"shard_id"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		Status    string `json:"status"`
		UserCount int    `json:"user_count"`
	}

	var statuses []ShardStatus

	for _, shard := range g.shards {
		status := ShardStatus{
			ShardID: shard.ShardID,
			Host:    shard.Host,
			Port:    shard.Port,
			Status:  "unknown",
		}

		healthURL := fmt.Sprintf("http://%s:%d/health", shard.Host, shard.Port)
		_, statusCode, err := g.forwardRequest("GET", healthURL, nil)
		if err != nil || statusCode != http.StatusOK {
			status.Status = "offline"
		} else {
			status.Status = "online"

			usersURL := fmt.Sprintf("http://%s:%d/users", shard.Host, shard.Port)
			respBody, _, err := g.forwardRequest("GET", usersURL, nil)
			if err == nil {
				var resp map[string]interface{}
				if json.Unmarshal(respBody, &resp) == nil {
					if count, ok := resp["count"].(float64); ok {
						status.UserCount = int(count)
					}
				}
			}
		}

		statuses = append(statuses, status)
	}

	c.JSON(http.StatusOK, gin.H{
		"shards":       statuses,
		"total_shards": len(g.shards),
	})
}

// HealthCheck returns gateway health
func (g *Gateway) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "api-gateway",
		"shards":  len(g.shards),
	})
}

func main() {
	port := flag.Int("port", 8000, "Gateway port")
	node1 := flag.String("node1", "localhost:8001", "Node 1 address")
	node2 := flag.String("node2", "localhost:8002", "Node 2 address")
	node3 := flag.String("node3", "localhost:8003", "Node 3 address")
	node4 := flag.String("node4", "localhost:8004", "Node 4 address")
	flag.Parse()

	parseAddr := func(addr string, shardID int) ShardConfig {
		var host string
		var p int
		fmt.Sscanf(addr, "%[^:]:%d", &host, &p)
		if host == "" {
			host = "localhost"
		}
		if p == 0 {
			p = 8000 + shardID
		}
		return ShardConfig{ShardID: shardID, Host: host, Port: p}
	}

	shards := []ShardConfig{
		parseAddr(*node1, 1),
		parseAddr(*node2, 2),
		parseAddr(*node3, 3),
		parseAddr(*node4, 4),
	}

	gateway := NewGateway(shards)

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

	router.GET("/health", gateway.HealthCheck)
	router.POST("/users", gateway.InsertUser)
	router.GET("/users/:id", gateway.GetUser)
	router.GET("/users", gateway.GetAllUsers)
	router.GET("/shards", gateway.GetShardStatus)

	log.Printf("Starting API Gateway on port %d", *port)
	log.Printf("Shard nodes: %v", shards)

	addr := fmt.Sprintf(":%d", *port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}
}
