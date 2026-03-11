package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"distributed-sharding/algorithms"

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

// ShardNode represents a single shard node server with algorithm support
type ShardNode struct {
	db          *sql.DB
	shardID     int
	port        int
	vectorClock *algorithms.VectorClock
	eventLog    *algorithms.EventLog
	snapshotMgr *algorithms.SnapshotManager
	election    *algorithms.BullyElection
	peerNodes   map[int]string // nodeID → "host:port"
	httpClient  *http.Client
}

// NewShardNode creates a new shard node with algorithm support
func NewShardNode(shardID, port int, dataDir string, totalNodes int) (*ShardNode, error) {
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

	// Setup vector clock
	nodeIDs := make([]string, totalNodes)
	allNodeInts := make([]int, totalNodes)
	for i := 0; i < totalNodes; i++ {
		nodeIDs[i] = fmt.Sprintf("node%d", i+1)
		allNodeInts[i] = i + 1
	}
	nodeIDStr := fmt.Sprintf("node%d", shardID)
	vectorClock := algorithms.NewVectorClock(nodeIDStr, nodeIDs)
	eventLog := algorithms.NewEventLog(nodeIDStr, vectorClock)

	node := &ShardNode{
		db:          db,
		shardID:     shardID,
		port:        port,
		vectorClock: vectorClock,
		eventLog:    eventLog,
		peerNodes:   make(map[int]string),
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}

	// Setup snapshot manager
	peerIDs := make([]string, 0)
	for i := 1; i <= totalNodes; i++ {
		if i != shardID {
			peerIDs = append(peerIDs, fmt.Sprintf("node%d", i))
		}
	}
	node.snapshotMgr = algorithms.NewSnapshotManager(nodeIDStr, peerIDs, vectorClock, node.getLocalState)

	// Setup leader election (sendFunc set later after peer addresses are known)
	sendFunc := func(toNodeID int, msg algorithms.ElectionMessage) bool {
		addr, ok := node.peerNodes[toNodeID]
		if !ok {
			return false
		}
		body, _ := json.Marshal(msg)
		resp, err := node.httpClient.Post(
			fmt.Sprintf("http://%s/election/message", addr),
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}
	node.election = algorithms.NewBullyElection(shardID, allNodeInts, sendFunc)

	return node, nil
}

// SetPeerAddresses sets the addresses of peer nodes for inter-node communication
func (n *ShardNode) SetPeerAddresses(peers map[int]string) {
	n.peerNodes = peers
}

// getLocalState returns the current state of this shard (for snapshots)
func (n *ShardNode) getLocalState() map[string]interface{} {
	var count int
	n.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)

	rows, err := n.db.Query("SELECT id, name, email FROM users")
	if err != nil {
		return map[string]interface{}{
			"user_count": count,
			"error":      err.Error(),
		}
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err == nil {
			users = append(users, u)
		}
	}

	return map[string]interface{}{
		"shard_id":   n.shardID,
		"user_count": count,
		"users":      users,
	}
}

// =============================================
// Original CRUD Handlers (with vector clock integration)
// =============================================

// InsertUser inserts a user into this shard (ticks vector clock)
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

	// Log event with vector clock
	n.eventLog.LogEvent("INSERT_USER", fmt.Sprintf("Created user %d (%s)", req.ID, req.Name))

	// Record message for any active snapshots
	n.snapshotMgr.RecordMessage(fmt.Sprintf("node%d", n.shardID),
		fmt.Sprintf("INSERT user %d", req.ID))

	c.JSON(http.StatusCreated, gin.H{
		"message":      "User created successfully",
		"user":         User{ID: req.ID, Name: req.Name, Email: req.Email},
		"shard_id":     n.shardID,
		"vector_clock": n.vectorClock.GetClock(),
	})
}

// GetUser retrieves a user from this shard (ticks vector clock)
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

	// Log event with vector clock
	n.eventLog.LogEvent("READ_USER", fmt.Sprintf("Read user %d (%s)", user.ID, user.Name))

	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"shard_id":     n.shardID,
		"vector_clock": n.vectorClock.GetClock(),
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

// =============================================
// Vector Clock Endpoints
// =============================================

// GetClock returns the current vector clock state and event log
func (n *ShardNode) GetClock(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"node_id":      fmt.Sprintf("node%d", n.shardID),
		"vector_clock": n.vectorClock.GetClock(),
		"events":       n.eventLog.GetEvents(),
		"event_count":  len(n.eventLog.GetEvents()),
	})
}

// PostClockEvent logs a custom event and ticks the vector clock
func (n *ShardNode) PostClockEvent(c *gin.Context) {
	var req struct {
		EventType   string `json:"event_type" binding:"required"`
		Description string `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	n.eventLog.LogEvent(req.EventType, req.Description)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Event logged",
		"vector_clock": n.vectorClock.GetClock(),
	})
}

// =============================================
// Chandy-Lamport Snapshot Endpoints
// =============================================

// InitiateSnapshot starts a new snapshot from this node
func (n *ShardNode) InitiateSnapshot(c *gin.Context) {
	var req struct {
		SnapshotID string `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SnapshotID == "" {
		req.SnapshotID = fmt.Sprintf("snap-%d-%d", n.shardID, time.Now().UnixMilli())
	}

	n.eventLog.LogEvent("SNAPSHOT_INITIATE", fmt.Sprintf("Initiating snapshot %s", req.SnapshotID))

	peers, err := n.snapshotMgr.InitiateSnapshot(req.SnapshotID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send markers to all peers
	markersSent := 0
	for _, peerIDStr := range peers {
		// Extract node number from "nodeX"
		var peerNum int
		fmt.Sscanf(peerIDStr, "node%d", &peerNum)

		addr, ok := n.peerNodes[peerNum]
		if !ok {
			continue
		}

		markerBody, _ := json.Marshal(gin.H{
			"snapshot_id": req.SnapshotID,
			"from_node":   fmt.Sprintf("node%d", n.shardID),
		})

		go func(address string) {
			n.httpClient.Post(
				fmt.Sprintf("http://%s/snapshot/marker", address),
				"application/json",
				bytes.NewBuffer(markerBody),
			)
		}(addr)
		markersSent++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Snapshot initiated",
		"snapshot_id":  req.SnapshotID,
		"markers_sent": markersSent,
		"initiated_by": fmt.Sprintf("node%d", n.shardID),
	})
}

// HandleMarker receives a snapshot marker from another node
func (n *ShardNode) HandleMarker(c *gin.Context) {
	var req struct {
		SnapshotID string `json:"snapshot_id" binding:"required"`
		FromNode   string `json:"from_node" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	n.eventLog.LogEvent("SNAPSHOT_MARKER_RECV",
		fmt.Sprintf("Received marker for %s from %s", req.SnapshotID, req.FromNode))

	peersToNotify, isFirst := n.snapshotMgr.HandleMarker(req.SnapshotID, req.FromNode)

	if isFirst {
		// Forward markers to all peers
		for _, peerIDStr := range peersToNotify {
			var peerNum int
			fmt.Sscanf(peerIDStr, "node%d", &peerNum)

			addr, ok := n.peerNodes[peerNum]
			if !ok {
				continue
			}

			markerBody, _ := json.Marshal(gin.H{
				"snapshot_id": req.SnapshotID,
				"from_node":   fmt.Sprintf("node%d", n.shardID),
			})

			go func(address string) {
				n.httpClient.Post(
					fmt.Sprintf("http://%s/snapshot/marker", address),
					"application/json",
					bytes.NewBuffer(markerBody),
				)
			}(addr)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Marker processed",
		"snapshot_id":   req.SnapshotID,
		"first_marker":  isFirst,
		"node_id":       fmt.Sprintf("node%d", n.shardID),
	})
}

// GetSnapshotState returns the snapshot state for a given ID
func (n *ShardNode) GetSnapshotState(c *gin.Context) {
	snapshotID := c.Query("id")
	if snapshotID == "" {
		// Return all snapshots
		c.JSON(http.StatusOK, gin.H{
			"node_id":   fmt.Sprintf("node%d", n.shardID),
			"snapshots": n.snapshotMgr.GetAllSnapshots(),
		})
		return
	}

	state, err := n.snapshotMgr.GetSnapshotState(snapshotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"node_id":  fmt.Sprintf("node%d", n.shardID),
		"snapshot": state,
	})
}

// =============================================
// Leader Election Endpoints
// =============================================

// StartElection triggers a Bully election from this node
func (n *ShardNode) StartElection(c *gin.Context) {
	n.eventLog.LogEvent("ELECTION_START", fmt.Sprintf("Node %d starting election", n.shardID))

	go n.election.StartElection()

	c.JSON(http.StatusOK, gin.H{
		"message": "Election started",
		"node_id": n.shardID,
		"state":   n.election.GetState(),
	})
}

// HandleElectionMessage processes an incoming election message
func (n *ShardNode) HandleElectionMessage(c *gin.Context) {
	var msg algorithms.ElectionMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch msg.Type {
	case algorithms.ELECTION:
		shouldStart := n.election.HandleElectionMessage(msg.FromNode, msg.Term)
		n.eventLog.LogEvent("ELECTION_MSG_RECV",
			fmt.Sprintf("Received ELECTION from node %d (term %d)", msg.FromNode, msg.Term))

		if shouldStart {
			go n.election.StartElection()
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "OK",
			"from_node":  n.shardID,
			"type":       "OK",
		})

	case algorithms.VICTORY:
		n.election.HandleVictoryMessage(msg.FromNode, msg.Term)
		n.eventLog.LogEvent("ELECTION_VICTORY_RECV",
			fmt.Sprintf("Received VICTORY from node %d (term %d)", msg.FromNode, msg.Term))

		c.JSON(http.StatusOK, gin.H{
			"message":    "Victory acknowledged",
			"new_leader": msg.FromNode,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown message type"})
	}
}

// GetElectionLeader returns the current election state
func (n *ShardNode) GetElectionLeader(c *gin.Context) {
	state := n.election.GetState()
	c.JSON(http.StatusOK, gin.H{
		"node_id": n.shardID,
		"state":   state,
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
	totalNodes := flag.Int("nodes", 4, "Total number of nodes in the system")
	peers := flag.String("peers", "", "Comma-separated peer addresses (e.g., 2=localhost:8002,3=localhost:8003)")
	flag.Parse()

	if *shardID < 1 || *shardID > *totalNodes {
		log.Fatalf("Invalid shard ID: %d. Must be between 1 and %d", *shardID, *totalNodes)
	}

	node, err := NewShardNode(*shardID, *port, *dataDir, *totalNodes)
	if err != nil {
		log.Fatalf("Failed to create shard node: %v", err)
	}
	defer node.Close()

	// Parse and set peer addresses
	if *peers != "" {
		peerMap := make(map[int]string)
		for _, p := range splitPeers(*peers) {
			peerMap[p.id] = p.addr
		}
		node.SetPeerAddresses(peerMap)
	} else {
		// Default: assume all nodes on localhost
		peerMap := make(map[int]string)
		for i := 1; i <= *totalNodes; i++ {
			if i != *shardID {
				peerMap[i] = fmt.Sprintf("localhost:%d", 8000+i)
			}
		}
		node.SetPeerAddresses(peerMap)
	}

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

	// Original endpoints
	router.GET("/health", node.HealthCheck)
	router.POST("/insert", node.InsertUser)
	router.GET("/user/:id", node.GetUser)
	router.GET("/users", node.GetAllUsers)

	// Vector Clock endpoints
	router.GET("/clock", node.GetClock)
	router.POST("/clock/event", node.PostClockEvent)

	// Chandy-Lamport Snapshot endpoints
	router.POST("/snapshot/initiate", node.InitiateSnapshot)
	router.POST("/snapshot/marker", node.HandleMarker)
	router.GET("/snapshot/state", node.GetSnapshotState)

	// Leader Election endpoints
	router.POST("/election/start", node.StartElection)
	router.POST("/election/message", node.HandleElectionMessage)
	router.GET("/election/leader", node.GetElectionLeader)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting Shard %d server on port %d (with algorithms: VectorClock, Snapshot, Election)", *shardID, *port)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// peerEntry holds a parsed peer address
type peerEntry struct {
	id   int
	addr string
}

// splitPeers parses "2=localhost:8002,3=localhost:8003" format
func splitPeers(s string) []peerEntry {
	var entries []peerEntry
	for _, part := range splitString(s, ',') {
		eqIdx := -1
		for i, c := range part {
			if c == '=' {
				eqIdx = i
				break
			}
		}
		if eqIdx == -1 {
			continue
		}
		idStr := part[:eqIdx]
		addr := part[eqIdx+1:]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		entries = append(entries, peerEntry{id: id, addr: addr})
	}
	return entries
}

// splitString splits a string by a separator rune
func splitString(s string, sep rune) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == sep {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
