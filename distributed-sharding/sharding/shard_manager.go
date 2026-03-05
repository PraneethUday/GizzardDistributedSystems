package sharding

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// ShardManager manages connections to multiple SQLite database shards
type ShardManager struct {
	shards    []*sql.DB
	numShards int
	mu        sync.RWMutex
}

// NewShardManager creates a new ShardManager with the specified number of shards
func NewShardManager(dataDir string, numShards int) (*ShardManager, error) {
	sm := &ShardManager{
		shards:    make([]*sql.DB, numShards),
		numShards: numShards,
	}

	// Initialize each shard database
	for i := 0; i < numShards; i++ {
		dbPath := fmt.Sprintf("%s/shard%d.db", dataDir, i+1)
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			// Close any already opened connections
			sm.Close()
			return nil, fmt.Errorf("failed to open shard %d: %w", i+1, err)
		}

		// Test the connection
		if err := db.Ping(); err != nil {
			sm.Close()
			return nil, fmt.Errorf("failed to ping shard %d: %w", i+1, err)
		}

		// Initialize the users table in this shard
		if err := initializeSchema(db); err != nil {
			sm.Close()
			return nil, fmt.Errorf("failed to initialize schema for shard %d: %w", i+1, err)
		}

		sm.shards[i] = db
		log.Printf("Initialized shard %d at %s", i+1, dbPath)
	}

	return sm, nil
}

// initializeSchema creates the users table if it doesn't exist
func initializeSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		name TEXT,
		email TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`
	_, err := db.Exec(schema)
	return err
}

// GetShard returns the database connection for the given userID
// Uses the formula: shard = (userID - 1) % numShards
// This ensures: user 1 → shard1.db, user 2 → shard2.db, etc.
func (sm *ShardManager) GetShard(userID int) *sql.DB {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Adjust userID to be 0-indexed for correct shard mapping
	shardIndex := (userID - 1) % sm.numShards
	if shardIndex < 0 {
		shardIndex = 0
	}
	return sm.shards[shardIndex]
}

// GetShardIndex returns the shard index (1-indexed) for the given userID
// user 1 → shard 1, user 2 → shard 2, user 5 → shard 1, etc.
func (sm *ShardManager) GetShardIndex(userID int) int {
	shardIndex := (userID - 1) % sm.numShards
	if shardIndex < 0 {
		shardIndex = 0
	}
	return shardIndex + 1 // 1-indexed for display
}

// GetAllShards returns all shard connections
func (sm *ShardManager) GetAllShards() []*sql.DB {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.shards
}

// GetNumShards returns the number of shards
func (sm *ShardManager) GetNumShards() int {
	return sm.numShards
}

// Close closes all shard connections
func (sm *ShardManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var lastErr error
	for i, db := range sm.shards {
		if db != nil {
			if err := db.Close(); err != nil {
				lastErr = fmt.Errorf("failed to close shard %d: %w", i+1, err)
				log.Printf("Error closing shard %d: %v", i+1, err)
			}
		}
	}
	return lastErr
}

// GetShardStats returns statistics for each shard
func (sm *ShardManager) GetShardStats() ([]ShardStats, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make([]ShardStats, sm.numShards)
	for i, db := range sm.shards {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to get count for shard %d: %w", i+1, err)
		}
		stats[i] = ShardStats{
			ShardID:   i + 1,
			UserCount: count,
		}
	}
	return stats, nil
}

// ShardStats holds statistics for a single shard
type ShardStats struct {
	ShardID   int `json:"shard_id"`
	UserCount int `json:"user_count"`
}
