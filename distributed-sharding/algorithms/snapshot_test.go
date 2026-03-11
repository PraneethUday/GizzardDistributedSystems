package algorithms

import (
	"testing"
)

func makeSnapshotManager(nodeID string, peers []string) *SnapshotManager {
	allNodes := append([]string{nodeID}, peers...)
	clock := NewVectorClock(nodeID, allNodes)

	getState := func() map[string]interface{} {
		return map[string]interface{}{
			"user_count": 5,
			"users":      []string{"Alice", "Bob"},
		}
	}

	return NewSnapshotManager(nodeID, peers, clock, getState)
}

func TestInitiateSnapshot(t *testing.T) {
	sm := makeSnapshotManager("node1", []string{"node2", "node3"})

	peers, err := sm.InitiateSnapshot("snap-001")
	if err != nil {
		t.Fatalf("Failed to initiate snapshot: %v", err)
	}

	if len(peers) != 2 {
		t.Errorf("Expected 2 peers, got %d", len(peers))
	}

	// Should not allow duplicate snapshot ID
	_, err = sm.InitiateSnapshot("snap-001")
	if err == nil {
		t.Error("Expected error for duplicate snapshot ID")
	}
}

func TestInitiateSnapshotRecordsState(t *testing.T) {
	sm := makeSnapshotManager("node1", []string{"node2", "node3"})

	sm.InitiateSnapshot("snap-002")

	state, err := sm.GetSnapshotState("snap-002")
	if err != nil {
		t.Fatalf("Failed to get snapshot state: %v", err)
	}

	if state.NodeID != "node1" {
		t.Errorf("Expected node ID 'node1', got '%s'", state.NodeID)
	}

	if state.LocalState["user_count"] != 5 {
		t.Errorf("Expected user_count = 5, got %v", state.LocalState["user_count"])
	}
}

func TestHandleMarkerFirstTime(t *testing.T) {
	sm := makeSnapshotManager("node2", []string{"node1", "node3"})

	peersToNotify, isFirst := sm.HandleMarker("snap-001", "node1")

	if !isFirst {
		t.Error("Expected first marker to return isFirst=true")
	}
	if len(peersToNotify) != 2 {
		t.Errorf("Expected 2 peers to notify, got %d", len(peersToNotify))
	}

	// Verify state was recorded
	state, err := sm.GetSnapshotState("snap-001")
	if err != nil {
		t.Fatalf("Failed to get snapshot state: %v", err)
	}
	if state.NodeID != "node2" {
		t.Errorf("Expected recorded state for node2, got %s", state.NodeID)
	}
}

func TestHandleMarkerDuplicate(t *testing.T) {
	sm := makeSnapshotManager("node2", []string{"node1", "node3"})

	// First marker from node1
	sm.HandleMarker("snap-001", "node1")

	// Duplicate marker from node3
	peersToNotify, isFirst := sm.HandleMarker("snap-001", "node3")

	if isFirst {
		t.Error("Expected duplicate marker to return isFirst=false")
	}
	if peersToNotify != nil {
		t.Error("Expected nil peers for duplicate marker")
	}
}

func TestSnapshotCompletion(t *testing.T) {
	sm := makeSnapshotManager("node2", []string{"node1", "node3"})

	// First marker from node1 — starts recording on node3's channel
	sm.HandleMarker("snap-001", "node1")

	// Record a message from node3 (simulating in-flight message)
	sm.RecordMessage("node3", "some message")

	// Marker from node3 — closes node3's channel → snapshot complete
	sm.HandleMarker("snap-001", "node3")

	state, _ := sm.GetSnapshotState("snap-001")
	if !state.Completed {
		t.Error("Expected snapshot to be completed after all markers received")
	}

	// Check that the recorded message is in node3's channel state
	if len(state.ChannelStates["node3"]) != 1 {
		t.Errorf("Expected 1 recorded message from node3, got %d", len(state.ChannelStates["node3"]))
	}
}

func TestGetAllSnapshots(t *testing.T) {
	sm := makeSnapshotManager("node1", []string{"node2"})

	sm.InitiateSnapshot("snap-A")
	sm.InitiateSnapshot("snap-B")

	all := sm.GetAllSnapshots()
	if len(all) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(all))
	}
}

func TestSnapshotNotFound(t *testing.T) {
	sm := makeSnapshotManager("node1", []string{"node2"})

	_, err := sm.GetSnapshotState("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent snapshot")
	}
}
