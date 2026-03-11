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

	"distributed-sharding/algorithms"

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
	hashRing   *algorithms.ConsistentHashRing
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
	// Initialize consistent hash ring
	hashRing := algorithms.NewHashRing(150)
	for _, s := range shards {
		nodeID := fmt.Sprintf("shard%d", s.ShardID)
		addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
		hashRing.AddNode(nodeID, addr)
	}

	return &Gateway{
		shards: shards,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		numShards: len(shards),
		hashRing:  hashRing,
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

// =============================================
// Original CRUD Handlers
// =============================================

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

// =============================================
// Vector Clock Endpoints (Gateway → Nodes)
// =============================================

// GetAllClocks fetches vector clocks from all nodes
func (g *Gateway) GetAllClocks(c *gin.Context) {
	logGateway("VECTOR CLOCKS request - Fetching from %d nodes", len(g.shards))

	type clockResult struct {
		shardID int
		data    map[string]interface{}
		err     string
	}

	results := make(chan clockResult, len(g.shards))

	for _, shard := range g.shards {
		go func(s ShardConfig) {
			url := fmt.Sprintf("http://%s:%d/clock", s.Host, s.Port)
			respBody, _, err := g.forwardRequest("GET", url, nil)
			if err != nil {
				results <- clockResult{shardID: s.ShardID, err: err.Error()}
				return
			}
			var data map[string]interface{}
			json.Unmarshal(respBody, &data)
			results <- clockResult{shardID: s.ShardID, data: data}
		}(shard)
	}

	clocks := make([]interface{}, 0)
	var errors []string
	for i := 0; i < len(g.shards); i++ {
		r := <-results
		if r.err != "" {
			errors = append(errors, fmt.Sprintf("shard %d: %s", r.shardID, r.err))
		} else {
			clocks = append(clocks, r.data)
		}
	}

	resp := gin.H{
		"clocks":      clocks,
		"node_count":  len(clocks),
		"description": "Lamport/Vector Clocks — each node maintains a vector of counters to track causal ordering of events across the distributed system",
	}
	if len(errors) > 0 {
		resp["errors"] = errors
	}
	c.JSON(http.StatusOK, resp)
}

// GetAllEvents fetches event logs from all nodes
func (g *Gateway) GetAllEvents(c *gin.Context) {
	logGateway("EVENT LOGS request - Fetching from %d nodes", len(g.shards))

	type eventResult struct {
		shardID int
		data    map[string]interface{}
		err     string
	}

	results := make(chan eventResult, len(g.shards))

	for _, shard := range g.shards {
		go func(s ShardConfig) {
			url := fmt.Sprintf("http://%s:%d/clock", s.Host, s.Port)
			respBody, _, err := g.forwardRequest("GET", url, nil)
			if err != nil {
				results <- eventResult{shardID: s.ShardID, err: err.Error()}
				return
			}
			var data map[string]interface{}
			json.Unmarshal(respBody, &data)
			results <- eventResult{shardID: s.ShardID, data: data}
		}(shard)
	}

	var allEvents []interface{}
	var errors []string
	for i := 0; i < len(g.shards); i++ {
		r := <-results
		if r.err != "" {
			errors = append(errors, fmt.Sprintf("shard %d: %s", r.shardID, r.err))
		} else if events, ok := r.data["events"].([]interface{}); ok {
			allEvents = append(allEvents, events...)
		}
	}

	resp := gin.H{
		"events":      allEvents,
		"event_count": len(allEvents),
	}
	if len(errors) > 0 {
		resp["errors"] = errors
	}
	c.JSON(http.StatusOK, resp)
}

// =============================================
// Chandy-Lamport Snapshot Endpoints (Gateway → Nodes)
// =============================================

// InitiateSnapshot triggers a snapshot across all nodes
func (g *Gateway) InitiateSnapshot(c *gin.Context) {
	logGateway("SNAPSHOT INITIATE request")

	var req struct {
		SnapshotID string `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SnapshotID == "" {
		req.SnapshotID = fmt.Sprintf("snap-gw-%d", time.Now().UnixMilli())
	}

	// Initiate from shard 1 (or first available shard)
	shard := g.shards[0]
	url := fmt.Sprintf("http://%s:%d/snapshot/initiate", shard.Host, shard.Port)
	body, _ := json.Marshal(gin.H{"snapshot_id": req.SnapshotID})
	respBody, _, err := g.forwardRequest("POST", url, body)

	if err != nil {
		logError("Failed to initiate snapshot: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	logSuccess("Snapshot %s initiated from Shard %d", req.SnapshotID, shard.ShardID)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Snapshot initiated",
		"snapshot_id":  req.SnapshotID,
		"initiated_at": shard.ShardID,
		"result":       result,
		"description":  "Chandy-Lamport Snapshot — captures a consistent global snapshot across all nodes without stopping the system",
	})
}

// GetSnapshotResults collects snapshot results from all nodes
func (g *Gateway) GetSnapshotResults(c *gin.Context) {
	snapshotID := c.Query("id")
	logGateway("SNAPSHOT RESULTS request - ID: %s", snapshotID)

	type snapshotResult struct {
		shardID int
		data    map[string]interface{}
		err     string
	}

	results := make(chan snapshotResult, len(g.shards))

	for _, shard := range g.shards {
		go func(s ShardConfig) {
			url := fmt.Sprintf("http://%s:%d/snapshot/state", s.Host, s.Port)
			if snapshotID != "" {
				url += "?id=" + snapshotID
			}
			respBody, _, err := g.forwardRequest("GET", url, nil)
			if err != nil {
				results <- snapshotResult{shardID: s.ShardID, err: err.Error()}
				return
			}
			var data map[string]interface{}
			json.Unmarshal(respBody, &data)
			results <- snapshotResult{shardID: s.ShardID, data: data}
		}(shard)
	}

	snapshots := make([]interface{}, 0)
	var errors []string
	for i := 0; i < len(g.shards); i++ {
		r := <-results
		if r.err != "" {
			errors = append(errors, fmt.Sprintf("shard %d: %s", r.shardID, r.err))
		} else {
			snapshots = append(snapshots, r.data)
		}
	}

	resp := gin.H{
		"snapshots":  snapshots,
		"node_count": len(snapshots),
	}
	if snapshotID != "" {
		resp["snapshot_id"] = snapshotID
	}
	if len(errors) > 0 {
		resp["errors"] = errors
	}
	c.JSON(http.StatusOK, resp)
}

// =============================================
// Leader Election Endpoints (Gateway → Nodes)
// =============================================

// TriggerElection starts a leader election from the lowest numbered node
func (g *Gateway) TriggerElection(c *gin.Context) {
	logGateway("LEADER ELECTION request - Triggering election")

	var req struct {
		FromNode int `json:"from_node"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FromNode == 0 {
		req.FromNode = 1 // Default: start from node 1
	}

	// Find the shard to start election from
	var targetShard ShardConfig
	for _, s := range g.shards {
		if s.ShardID == req.FromNode {
			targetShard = s
			break
		}
	}

	url := fmt.Sprintf("http://%s:%d/election/start", targetShard.Host, targetShard.Port)
	respBody, _, err := g.forwardRequest("POST", url, nil)

	if err != nil {
		logError("Failed to trigger election: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	logSuccess("Election triggered from Node %d", req.FromNode)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Election triggered",
		"from_node":    req.FromNode,
		"result":       result,
		"description":  "Bully Algorithm — the node with the highest ID always wins the election",
	})
}

// GetLeaderStatus fetches leader info from all nodes
func (g *Gateway) GetLeaderStatus(c *gin.Context) {
	logGateway("LEADER STATUS request - Querying %d nodes", len(g.shards))

	type leaderResult struct {
		shardID int
		data    map[string]interface{}
		err     string
	}

	results := make(chan leaderResult, len(g.shards))

	for _, shard := range g.shards {
		go func(s ShardConfig) {
			url := fmt.Sprintf("http://%s:%d/election/leader", s.Host, s.Port)
			respBody, _, err := g.forwardRequest("GET", url, nil)
			if err != nil {
				results <- leaderResult{shardID: s.ShardID, err: err.Error()}
				return
			}
			var data map[string]interface{}
			json.Unmarshal(respBody, &data)
			results <- leaderResult{shardID: s.ShardID, data: data}
		}(shard)
	}

	nodeStates := make([]interface{}, 0)
	var errors []string
	for i := 0; i < len(g.shards); i++ {
		r := <-results
		if r.err != "" {
			errors = append(errors, fmt.Sprintf("shard %d: %s", r.shardID, r.err))
		} else {
			nodeStates = append(nodeStates, r.data)
		}
	}

	resp := gin.H{
		"nodes":      nodeStates,
		"node_count": len(nodeStates),
	}
	if len(errors) > 0 {
		resp["errors"] = errors
	}
	c.JSON(http.StatusOK, resp)
}

// =============================================
// Consistent Hashing Endpoints (Gateway-local)
// =============================================

// GetHashRingStatus returns the consistent hash ring state
func (g *Gateway) GetHashRingStatus(c *gin.Context) {
	logGateway("HASH RING STATUS request")

	status := g.hashRing.GetRingStatus()

	c.JSON(http.StatusOK, gin.H{
		"hash_ring":    status,
		"description":  "Consistent Hashing — uses a hash ring with virtual nodes to distribute keys, minimizing redistribution when nodes join/leave",
	})
}

// LookupHashRingKey looks up which node a key maps to on the hash ring
func (g *Gateway) LookupHashRingKey(c *gin.Context) {
	var req struct {
		Key    string `json:"key"`
		UserID int    `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := req.Key
	if key == "" && req.UserID > 0 {
		key = fmt.Sprintf("user_%d", req.UserID)
	}
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide either 'key' or 'user_id'"})
		return
	}

	details := g.hashRing.LookupKeyDetails(key)

	// Also show the modulo-based shard for comparison
	if req.UserID > 0 {
		modShard := (req.UserID-1)%g.numShards + 1
		details["modulo_shard"] = fmt.Sprintf("shard%d", modShard)
		details["comparison"] = fmt.Sprintf("Modulo: shard%d vs Consistent Hash: %s", modShard, details["assigned_node"])
	}

	c.JSON(http.StatusOK, details)
}

// AddHashRingNode adds a new node to the consistent hash ring
func (g *Gateway) AddHashRingNode(c *gin.Context) {
	var req struct {
		NodeID  string `json:"node_id" binding:"required"`
		Address string `json:"address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	g.hashRing.AddNode(req.NodeID, req.Address)
	logSuccess("Added node %s (%s) to hash ring", req.NodeID, req.Address)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Node %s added to hash ring", req.NodeID),
		"status":  g.hashRing.GetRingStatus(),
	})
}

// RemoveHashRingNode removes a node from the consistent hash ring
func (g *Gateway) RemoveHashRingNode(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Node ID required"})
		return
	}

	g.hashRing.RemoveNode(nodeID)
	logSuccess("Removed node %s from hash ring", nodeID)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Node %s removed from hash ring", nodeID),
		"status":  g.hashRing.GetRingStatus(),
	})
}

// =============================================
// Algorithms Overview Endpoint
// =============================================

// ListAlgorithms returns info about all implemented algorithms
func (g *Gateway) ListAlgorithms(c *gin.Context) {
	algorithms := []gin.H{
		{
			"name":        "Lamport/Vector Clocks",
			"description": "Each node maintains a vector of logical timestamps to capture causal ordering of events. If event A's vector clock ≤ event B's, then A happened-before B. If neither ≤ holds, the events are concurrent.",
			"endpoints": []string{
				"GET /clocks — fetch vector clocks from all nodes",
				"GET /events — fetch event logs with timestamps from all nodes",
			},
			"node_endpoints": []string{
				"GET /clock — this node's vector clock and event log",
				"POST /clock/event — log a custom event",
			},
		},
		{
			"name":        "Chandy-Lamport Snapshot Algorithm",
			"description": "Captures a consistent global snapshot of the distributed system without stopping it. Uses marker messages: when a node receives its first marker, it records its local state and forwards markers. Channel states are recorded between marker arrivals.",
			"endpoints": []string{
				"POST /snapshot — initiate a global snapshot",
				"GET /snapshot — collect snapshot results from all nodes",
			},
			"node_endpoints": []string{
				"POST /snapshot/initiate — start a snapshot from this node",
				"POST /snapshot/marker — receive a marker from a peer",
				"GET /snapshot/state — get snapshot state",
			},
		},
		{
			"name":        "Bully Leader Election Algorithm",
			"description": "Elects a leader (coordinator) among nodes. When a node detects the leader is down, it sends ELECTION messages to all higher-ID nodes. If no higher node responds, it declares itself leader via VICTORY messages. The highest-ID alive node always wins.",
			"endpoints": []string{
				"POST /election/start — trigger a leader election",
				"GET /election/leader — get leader status from all nodes",
			},
			"node_endpoints": []string{
				"POST /election/start — trigger election from this node",
				"POST /election/message — receive election/victory message",
				"GET /election/leader — get this node's view of the leader",
			},
		},
		{
			"name":        "Consistent Hashing",
			"description": "Distributes keys across nodes using a hash ring with virtual nodes. Each node is mapped to multiple positions on the ring (virtual nodes). To find a key's owner, hash the key and walk clockwise to the first node. Adding/removing nodes only affects adjacent segments, minimizing data movement.",
			"endpoints": []string{
				"GET /hash-ring/status — view hash ring state and key distribution",
				"POST /hash-ring/lookup — lookup which node owns a given key",
				"POST /hash-ring/add-node — add a node to the ring",
				"DELETE /hash-ring/remove-node/:id — remove a node from the ring",
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"algorithms":       algorithms,
		"total_algorithms": len(algorithms),
		"project":          "GizzardDistributedSystems",
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

	// Original endpoints
	router.GET("/health", gateway.HealthCheck)
	router.POST("/users", gateway.InsertUser)
	router.GET("/users/:id", gateway.GetUser)
	router.GET("/users", gateway.GetAllUsers)
	router.GET("/shards", gateway.GetShardStatus)

	// Algorithm overview
	router.GET("/algorithms", gateway.ListAlgorithms)

	// Vector Clock endpoints
	router.GET("/clocks", gateway.GetAllClocks)
	router.GET("/events", gateway.GetAllEvents)

	// Chandy-Lamport Snapshot endpoints
	router.POST("/snapshot", gateway.InitiateSnapshot)
	router.GET("/snapshot", gateway.GetSnapshotResults)

	// Leader Election endpoints
	router.POST("/election/start", gateway.TriggerElection)
	router.GET("/election/leader", gateway.GetLeaderStatus)

	// Consistent Hashing endpoints
	router.GET("/hash-ring/status", gateway.GetHashRingStatus)
	router.POST("/hash-ring/lookup", gateway.LookupHashRingKey)
	router.POST("/hash-ring/add-node", gateway.AddHashRingNode)
	router.DELETE("/hash-ring/remove-node/:id", gateway.RemoveHashRingNode)

	log.Printf("Starting API Gateway on port %d", *port)
	log.Printf("Shard nodes: %v", shards)
	log.Println("Algorithms: VectorClock, Chandy-Lamport Snapshot, Bully Election, Consistent Hashing")
	log.Println("Endpoints: GET /algorithms for full listing")

	addr := fmt.Sprintf(":%d", *port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}
}
