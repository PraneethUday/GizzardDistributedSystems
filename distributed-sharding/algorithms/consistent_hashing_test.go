package algorithms

import (
	"fmt"
	"testing"
)

func TestNewHashRing(t *testing.T) {
	ring := NewHashRing(100)
	if ring.NodeCount() != 0 {
		t.Errorf("Expected 0 nodes in new ring, got %d", ring.NodeCount())
	}
}

func TestAddNode(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("shard1", "localhost:8001")
	ring.AddNode("shard2", "localhost:8002")

	if ring.NodeCount() != 2 {
		t.Errorf("Expected 2 nodes, got %d", ring.NodeCount())
	}

	nodes := ring.GetNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes from GetNodes, got %d", len(nodes))
	}
}

func TestGetNode(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("shard1", "localhost:8001")
	ring.AddNode("shard2", "localhost:8002")
	ring.AddNode("shard3", "localhost:8003")

	// Any key should map to one of the three nodes
	validNodes := map[string]bool{"shard1": true, "shard2": true, "shard3": true}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("user_%d", i)
		node, err := ring.GetNode(key)
		if err != nil {
			t.Fatalf("GetNode failed for key %s: %v", key, err)
		}
		if !validNodes[node] {
			t.Errorf("GetNode returned invalid node %s for key %s", node, key)
		}
	}
}

func TestGetNodeConsistency(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("shard1", "localhost:8001")
	ring.AddNode("shard2", "localhost:8002")

	// Same key should always map to same node
	node1, _ := ring.GetNode("test_key")
	node2, _ := ring.GetNode("test_key")
	if node1 != node2 {
		t.Error("Same key mapped to different nodes!")
	}
}

func TestGetNodeEmptyRing(t *testing.T) {
	ring := NewHashRing(100)
	_, err := ring.GetNode("some_key")
	if err == nil {
		t.Error("Expected error when getting node from empty ring")
	}
}

func TestRemoveNode(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("shard1", "localhost:8001")
	ring.AddNode("shard2", "localhost:8002")
	ring.AddNode("shard3", "localhost:8003")

	ring.RemoveNode("shard2")

	if ring.NodeCount() != 2 {
		t.Errorf("Expected 2 nodes after removal, got %d", ring.NodeCount())
	}

	// All keys should now map to shard1 or shard3
	for i := 0; i < 50; i++ {
		node, _ := ring.GetNode(fmt.Sprintf("key_%d", i))
		if node != "shard1" && node != "shard3" {
			t.Errorf("After removing shard2, key mapped to %s", node)
		}
	}
}

func TestMinimalRedistribution(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("shard1", "localhost:8001")
	ring.AddNode("shard2", "localhost:8002")
	ring.AddNode("shard3", "localhost:8003")

	// Record where 1000 keys land
	before := make(map[string]string)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user_%d", i)
		node, _ := ring.GetNode(key)
		before[key] = node
	}

	// Add a new node
	ring.AddNode("shard4", "localhost:8004")

	// Check how many keys moved
	moved := 0
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("user_%d", i)
		node, _ := ring.GetNode(key)
		if node != before[key] {
			moved++
		}
	}

	// With consistent hashing, roughly 1/N keys should move (where N is new total)
	// Allow generous margin: at most 50% keys should move
	if moved > 500 {
		t.Errorf("Too many keys moved: %d/1000 (expected much fewer with consistent hashing)", moved)
	}

	t.Logf("Keys redistributed: %d/1000 (%.1f%%)", moved, float64(moved)/10.0)
}

func TestDistribution(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("shard1", "localhost:8001")
	ring.AddNode("shard2", "localhost:8002")
	ring.AddNode("shard3", "localhost:8003")
	ring.AddNode("shard4", "localhost:8004")

	// Check distribution across 10000 keys
	counts := make(map[string]int)
	total := 10000

	for i := 0; i < total; i++ {
		key := fmt.Sprintf("key_%d", i)
		node, _ := ring.GetNode(key)
		counts[node]++
	}

	// Each node should get roughly 25% (allow 10-40% range)
	for node, count := range counts {
		pct := float64(count) / float64(total) * 100
		t.Logf("%s: %d keys (%.1f%%)", node, count, pct)
		if pct < 10 || pct > 40 {
			t.Errorf("Uneven distribution: %s got %.1f%% (expected ~25%%)", node, pct)
		}
	}
}

func TestGetRingStatus(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("shard1", "localhost:8001")
	ring.AddNode("shard2", "localhost:8002")

	status := ring.GetRingStatus()

	if len(status.Nodes) != 2 {
		t.Errorf("Expected 2 nodes in status, got %d", len(status.Nodes))
	}
	if status.VirtualNodes != 100 {
		t.Errorf("Expected 100 virtual nodes per node, got %d", status.VirtualNodes)
	}
	if status.TotalVNodes != 200 {
		t.Errorf("Expected 200 total virtual nodes, got %d", status.TotalVNodes)
	}
	if len(status.Distribution) != 2 {
		t.Errorf("Expected distribution for 2 nodes, got %d", len(status.Distribution))
	}
}

func TestLookupKeyDetails(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("shard1", "localhost:8001")

	details := ring.LookupKeyDetails("user_42")

	if details["key"] != "user_42" {
		t.Errorf("Expected key 'user_42', got %v", details["key"])
	}
	if details["assigned_node"] != "shard1" {
		t.Errorf("Expected assigned_node 'shard1', got %v", details["assigned_node"])
	}
}

func TestGetNodeForUserID(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("shard1", "localhost:8001")
	ring.AddNode("shard2", "localhost:8002")

	node, err := ring.GetNodeForUserID(42)
	if err != nil {
		t.Fatalf("GetNodeForUserID failed: %v", err)
	}
	if node != "shard1" && node != "shard2" {
		t.Errorf("Unexpected node: %s", node)
	}
}

func TestDefaultReplicas(t *testing.T) {
	ring := NewHashRing(0) // should default to 150
	ring.AddNode("shard1", "localhost:8001")

	status := ring.GetRingStatus()
	if status.VirtualNodes != 150 {
		t.Errorf("Expected default 150 replicas, got %d", status.VirtualNodes)
	}
}
