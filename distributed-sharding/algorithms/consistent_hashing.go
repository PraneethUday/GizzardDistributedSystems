package algorithms

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
)

// HashRingNode represents a physical node on the consistent hash ring.
type HashRingNode struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	IsActive bool   `json:"is_active"`
}

// VirtualNode represents a virtual node mapped onto the hash ring.
// Multiple virtual nodes map to the same physical node for better distribution.
type VirtualNode struct {
	Hash     uint32 `json:"hash"`
	NodeID   string `json:"node_id"`
	VNodeIdx int    `json:"vnode_index"`
}

// HashRingStatus holds the full state of the consistent hash ring.
type HashRingStatus struct {
	Nodes         []HashRingNode       `json:"nodes"`
	VirtualNodes  int                  `json:"virtual_nodes_per_node"`
	TotalVNodes   int                  `json:"total_virtual_nodes"`
	RingSize      uint32               `json:"ring_size"`
	Distribution  map[string]float64   `json:"key_distribution_pct"`
}

// ConsistentHashRing implements consistent hashing with virtual nodes.
//
// How it works:
//  1. Each physical node is mapped to multiple positions on a hash ring
//     using hash(nodeID + replicaIndex). These are called "virtual nodes".
//  2. To find which node owns a key, we hash the key and walk clockwise
//     on the ring until we find the first virtual node. That virtual node's
//     physical node owns the key.
//  3. Adding/removing a node only affects keys in adjacent ring segments,
//     minimizing data movement compared to modulo-based sharding.
//
// Virtual nodes ensure uniform distribution even with few physical nodes.
type ConsistentHashRing struct {
	replicas     int                       // number of virtual nodes per physical node
	ring         []uint32                  // sorted hash values of virtual nodes
	hashMap      map[uint32]string         // hash → physical node ID
	nodes        map[string]*HashRingNode  // physical nodes
	mu           sync.RWMutex
}

// NewHashRing creates a new consistent hash ring.
// replicas is the number of virtual nodes per physical node (typically 100-300).
func NewHashRing(replicas int) *ConsistentHashRing {
	if replicas <= 0 {
		replicas = 150 // sensible default
	}
	return &ConsistentHashRing{
		replicas: replicas,
		ring:     make([]uint32, 0),
		hashMap:  make(map[uint32]string),
		nodes:    make(map[string]*HashRingNode),
	}
}

// AddNode adds a physical node to the hash ring with virtual replicas.
func (hr *ConsistentHashRing) AddNode(nodeID, address string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	hr.nodes[nodeID] = &HashRingNode{
		ID:       nodeID,
		Address:  address,
		IsActive: true,
	}

	for i := 0; i < hr.replicas; i++ {
		hash := hr.hashKey(fmt.Sprintf("%s#%d", nodeID, i))
		hr.ring = append(hr.ring, hash)
		hr.hashMap[hash] = nodeID
	}

	sort.Slice(hr.ring, func(i, j int) bool {
		return hr.ring[i] < hr.ring[j]
	})
}

// RemoveNode removes a physical node and all its virtual nodes from the ring.
func (hr *ConsistentHashRing) RemoveNode(nodeID string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	delete(hr.nodes, nodeID)

	// Remove all virtual nodes for this physical node
	newRing := make([]uint32, 0, len(hr.ring))
	for _, hash := range hr.ring {
		if hr.hashMap[hash] != nodeID {
			newRing = append(newRing, hash)
		} else {
			delete(hr.hashMap, hash)
		}
	}
	hr.ring = newRing
}

// GetNode returns the physical node ID responsible for the given key.
// It hashes the key and finds the first virtual node clockwise on the ring.
func (hr *ConsistentHashRing) GetNode(key string) (string, error) {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	if len(hr.ring) == 0 {
		return "", fmt.Errorf("hash ring is empty — no nodes available")
	}

	hash := hr.hashKey(key)

	// Binary search for the first virtual node with hash >= key hash
	idx := sort.Search(len(hr.ring), func(i int) bool {
		return hr.ring[i] >= hash
	})

	// Wrap around to the beginning of the ring
	if idx == len(hr.ring) {
		idx = 0
	}

	return hr.hashMap[hr.ring[idx]], nil
}

// GetNodeForUserID returns the node for a given user ID (convenience method).
func (hr *ConsistentHashRing) GetNodeForUserID(userID int) (string, error) {
	return hr.GetNode(fmt.Sprintf("user_%d", userID))
}

// GetNodes returns all physical nodes.
func (hr *ConsistentHashRing) GetNodes() []HashRingNode {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	nodes := make([]HashRingNode, 0, len(hr.nodes))
	for _, node := range hr.nodes {
		nodes = append(nodes, *node)
	}
	return nodes
}

// GetRingStatus returns the full state of the hash ring including
// estimated key distribution percentages.
func (hr *ConsistentHashRing) GetRingStatus() HashRingStatus {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	nodes := make([]HashRingNode, 0, len(hr.nodes))
	for _, node := range hr.nodes {
		nodes = append(nodes, *node)
	}

	distribution := hr.calculateDistribution()

	return HashRingStatus{
		Nodes:        nodes,
		VirtualNodes: hr.replicas,
		TotalVNodes:  len(hr.ring),
		RingSize:     ^uint32(0), // max uint32
		Distribution: distribution,
	}
}

// LookupKeyDetails returns the node and hash information for a given key.
func (hr *ConsistentHashRing) LookupKeyDetails(key string) map[string]interface{} {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	hash := hr.hashKey(key)
	nodeID := ""
	vnodeHash := uint32(0)

	if len(hr.ring) > 0 {
		idx := sort.Search(len(hr.ring), func(i int) bool {
			return hr.ring[i] >= hash
		})
		if idx == len(hr.ring) {
			idx = 0
		}
		nodeID = hr.hashMap[hr.ring[idx]]
		vnodeHash = hr.ring[idx]
	}

	return map[string]interface{}{
		"key":            key,
		"key_hash":       hash,
		"assigned_node":  nodeID,
		"vnode_hash":     vnodeHash,
		"total_nodes":    len(hr.nodes),
		"total_vnodes":   len(hr.ring),
	}
}

// calculateDistribution estimates key distribution across nodes
// by measuring the proportion of the ring owned by each node.
func (hr *ConsistentHashRing) calculateDistribution() map[string]float64 {
	if len(hr.ring) == 0 {
		return nil
	}

	// Count ring segments owned by each node
	ownership := make(map[string]uint64)
	totalSpace := uint64(^uint32(0)) + 1 // full ring size = 2^32

	for i := 0; i < len(hr.ring); i++ {
		nodeID := hr.hashMap[hr.ring[i]]
		var segmentSize uint64
		if i == 0 {
			// Segment wraps around: from last vnode to first vnode
			segmentSize = uint64(hr.ring[0]) + (totalSpace - uint64(hr.ring[len(hr.ring)-1]))
		} else {
			segmentSize = uint64(hr.ring[i]) - uint64(hr.ring[i-1])
		}
		ownership[nodeID] += segmentSize
	}

	distribution := make(map[string]float64)
	for nodeID, owned := range ownership {
		distribution[nodeID] = float64(owned) / float64(totalSpace) * 100.0
	}

	return distribution
}

// hashKey produces a uint32 hash using SHA-256 (first 4 bytes).
func (hr *ConsistentHashRing) hashKey(key string) uint32 {
	h := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint32(h[:4])
}

// NodeCount returns the number of physical nodes.
func (hr *ConsistentHashRing) NodeCount() int {
	hr.mu.RLock()
	defer hr.mu.RUnlock()
	return len(hr.nodes)
}
