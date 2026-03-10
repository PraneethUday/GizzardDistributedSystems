package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Color definitions for shard logging
var (
	shardColors = []*color.Color{
		color.New(color.FgHiBlue, color.Bold),    // Shard 1 - Blue
		color.New(color.FgHiGreen, color.Bold),   // Shard 2 - Green
		color.New(color.FgHiYellow, color.Bold),  // Shard 3 - Yellow
		color.New(color.FgHiMagenta, color.Bold), // Shard 4 - Magenta
	}
	gatewayColor = color.New(color.FgHiCyan, color.Bold)
	successColor = color.New(color.FgGreen)
	errorColor   = color.New(color.FgRed, color.Bold)
	infoColor    = color.New(color.FgWhite)
)

// logShard logs a message with shard-specific formatting
func logShard(shardID int, format string, args ...interface{}) {
	colorIdx := (shardID - 1) % len(shardColors)
	prefix := shardColors[colorIdx].Sprintf("[SHARD %d]", shardID)
	message := fmt.Sprintf(format, args...)
	log.Printf("%s %s", prefix, message)
}

// logGateway logs a gateway-level message
func logGateway(format string, args ...interface{}) {
	prefix := gatewayColor.Sprint("[GATEWAY]")
	message := fmt.Sprintf(format, args...)
	log.Printf("%s %s", prefix, message)
}

// logSuccess logs a success message
func logSuccess(format string, args ...interface{}) {
	prefix := successColor.Sprint("[SUCCESS]")
	message := fmt.Sprintf(format, args...)
	log.Printf("%s %s", prefix, message)
}

// logError logs an error message
func logError(format string, args ...interface{}) {
	prefix := errorColor.Sprint("[ERROR]")
	message := fmt.Sprintf(format, args...)
	log.Printf("%s %s", prefix, message)
}

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
			Timeout: 2 * time.Second, // Short timeout for faster failure detection
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
		logError("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	shard := g.GetShardForUser(req.ID)
	shardURL := fmt.Sprintf("http://%s:%d/insert", shard.Host, shard.Port)

	logGateway("CREATE USER request - ID: %d, Name: %s, Email: %s", req.ID, req.Name, req.Email)
	logShard(shard.ShardID, "Routing INSERT request to %s:%d", shard.Host, shard.Port)

	bodyBytes, _ := json.Marshal(req)
	startTime := time.Now()
	respBody, statusCode, err := g.forwardRequest("POST", shardURL, bodyBytes)
	duration := time.Since(startTime)

	if err != nil {
		logShard(shard.ShardID, "FAILED - Connection error: %v", err)
		logError("Shard %d unreachable after %v", shard.ShardID, duration)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "Failed to reach shard node",
			"details": err.Error(),
			"shard":   shard.ShardID,
		})
		return
	}

	logShard(shard.ShardID, "RESPONSE [%d] in %v", statusCode, duration)
	if statusCode == http.StatusOK || statusCode == http.StatusCreated {
		logSuccess("User %d created successfully on Shard %d", req.ID, shard.ShardID)
	}

	c.Data(statusCode, "application/json", respBody)
}

// GetUser handles GET /users/:id
func (g *Gateway) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logError("Invalid user ID: %s", idStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	shard := g.GetShardForUser(id)
	shardURL := fmt.Sprintf("http://%s:%d/user/%d", shard.Host, shard.Port, id)

	logGateway("FETCH USER request - ID: %d", id)
	logShard(shard.ShardID, "Routing GET request to %s:%d", shard.Host, shard.Port)

	startTime := time.Now()
	respBody, statusCode, err := g.forwardRequest("GET", shardURL, nil)
	duration := time.Since(startTime)

	if err != nil {
		logShard(shard.ShardID, "FAILED - Connection error: %v", err)
		logError("Shard %d unreachable after %v", shard.ShardID, duration)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "Failed to reach shard node",
			"details": err.Error(),
			"shard":   shard.ShardID,
		})
		return
	}

	logShard(shard.ShardID, "RESPONSE [%d] in %v", statusCode, duration)
	if statusCode == http.StatusOK {
		logSuccess("User %d fetched from Shard %d", id, shard.ShardID)
	} else if statusCode == http.StatusNotFound {
		logShard(shard.ShardID, "User %d not found", id)
	}

	c.Data(statusCode, "application/json", respBody)
}

// GetAllUsers handles GET /users
func (g *Gateway) GetAllUsers(c *gin.Context) {
	logGateway("FETCH ALL USERS request - Querying %d shards in parallel", len(g.shards))

	type shardResult struct {
		shardID int
		users   []map[string]interface{}
		err     string
		count   int
	}

	results := make(chan shardResult, len(g.shards))
	startTime := time.Now()

	// Query all shards in parallel
	for _, shard := range g.shards {
		go func(s ShardConfig) {
			shardURL := fmt.Sprintf("http://%s:%d/users", s.Host, s.Port)
			logShard(s.ShardID, "Querying users at %s:%d", s.Host, s.Port)
			
			respBody, statusCode, err := g.forwardRequest("GET", shardURL, nil)
			if err != nil {
				logShard(s.ShardID, "FAILED - %v", err)
				results <- shardResult{shardID: s.ShardID, err: fmt.Sprintf("shard %d: %v", s.ShardID, err)}
				return
			}

			var users []map[string]interface{}
			if statusCode == http.StatusOK {
				var shardResp map[string]interface{}
				if err := json.Unmarshal(respBody, &shardResp); err == nil {
					if userList, ok := shardResp["users"].([]interface{}); ok {
						for _, u := range userList {
							if userMap, ok := u.(map[string]interface{}); ok {
								userMap["shard_id"] = s.ShardID
								users = append(users, userMap)
							}
						}
					}
				}
				logShard(s.ShardID, "RESPONSE [200] - Found %d users", len(users))
			}
			results <- shardResult{shardID: s.ShardID, users: users, count: len(users)}
		}(shard)
	}

	// Collect results
	var allUsers []map[string]interface{}
	var errors []string

	for i := 0; i < len(g.shards); i++ {
		result := <-results
		if result.err != "" {
			errors = append(errors, result.err)
		} else {
			allUsers = append(allUsers, result.users...)
		}
	}

	duration := time.Since(startTime)
	logSuccess("Fetched %d total users from %d shards in %v", len(allUsers), len(g.shards)-len(errors), duration)

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
	logGateway("HEALTH CHECK request - Checking %d shards", len(g.shards))

	type ShardStatus struct {
		ShardID   int    `json:"shard_id"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		Status    string `json:"status"`
		UserCount int    `json:"user_count"`
	}

	results := make(chan ShardStatus, len(g.shards))
	startTime := time.Now()

	// Check all shards in parallel
	for _, shard := range g.shards {
		go func(s ShardConfig) {
			status := ShardStatus{
				ShardID: s.ShardID,
				Host:    s.Host,
				Port:    s.Port,
				Status:  "offline",
			}

			logShard(s.ShardID, "Checking health at %s:%d", s.Host, s.Port)
			healthURL := fmt.Sprintf("http://%s:%d/health", s.Host, s.Port)
			_, statusCode, err := g.forwardRequest("GET", healthURL, nil)
			if err == nil && statusCode == http.StatusOK {
				status.Status = "online"
				logShard(s.ShardID, "ONLINE - Health check passed")

				usersURL := fmt.Sprintf("http://%s:%d/users", s.Host, s.Port)
				respBody, _, err := g.forwardRequest("GET", usersURL, nil)
				if err == nil {
					var resp map[string]interface{}
					if json.Unmarshal(respBody, &resp) == nil {
						if count, ok := resp["count"].(float64); ok {
							status.UserCount = int(count)
							logShard(s.ShardID, "Contains %d users", status.UserCount)
						}
					}
				}
			} else {
				logShard(s.ShardID, "OFFLINE - Not responding")
			}

			results <- status
		}(shard)
	}

	// Collect results and sort by shard ID
	statuses := make([]ShardStatus, len(g.shards))
	onlineCount := 0
	for i := 0; i < len(g.shards); i++ {
		status := <-results
		statuses[status.ShardID-1] = status
		if status.Status == "online" {
			onlineCount++
		}
	}

	duration := time.Since(startTime)
	logSuccess("Health check complete: %d/%d shards online in %v", onlineCount, len(g.shards), duration)

	c.JSON(http.StatusOK, gin.H{
		"shards":       statuses,
		"total_shards": len(g.shards),
	})
}

// HealthCheck returns gateway health
func (g *Gateway) HealthCheck(c *gin.Context) {
	logGateway("Gateway health check - OK")
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "api-gateway",
		"shards":  len(g.shards),
	})
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults and command-line flags")
	}

	// Helper function to get env with fallback
	getEnv := func(key, fallback string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return fallback
	}
	getEnvInt := func(key string, fallback int) int {
		if value := os.Getenv(key); value != "" {
			if num, err := strconv.Atoi(value); err == nil {
				return num
			}
		}
		return fallback
	}

	// Build default addresses from environment variables
	defaultNode1 := fmt.Sprintf("%s:%d", getEnv("SHARD1_HOST", "localhost"), getEnvInt("SHARD1_PORT", 8001))
	defaultNode2 := fmt.Sprintf("%s:%d", getEnv("SHARD2_HOST", "localhost"), getEnvInt("SHARD2_PORT", 8002))
	defaultNode3 := fmt.Sprintf("%s:%d", getEnv("SHARD3_HOST", "localhost"), getEnvInt("SHARD3_PORT", 8003))
	defaultNode4 := fmt.Sprintf("%s:%d", getEnv("SHARD4_HOST", "localhost"), getEnvInt("SHARD4_PORT", 8004))

	port := flag.Int("port", getEnvInt("GATEWAY_PORT", 8000), "Gateway port")
	node1 := flag.String("node1", defaultNode1, "Node 1 address")
	node2 := flag.String("node2", defaultNode2, "Node 2 address")
	node3 := flag.String("node3", defaultNode3, "Node 3 address")
	node4 := flag.String("node4", defaultNode4, "Node 4 address")
	flag.Parse()

	parseAddr := func(addr string, shardID int) ShardConfig {
		host := "localhost"
		p := 8000 + shardID
		
		// Split on the last colon to handle IPv4 addresses like 192.168.1.1:8003
		lastColon := strings.LastIndex(addr, ":")
		if lastColon != -1 {
			host = addr[:lastColon]
			if portNum, err := strconv.Atoi(addr[lastColon+1:]); err == nil {
				p = portNum
			}
		}
		
		if host == "" {
			host = "localhost"
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
