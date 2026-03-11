package algorithms

import (
	"fmt"
	"sync"
	"time"
)

// SnapshotState holds the local state captured during a Chandy-Lamport snapshot.
type SnapshotState struct {
	SnapshotID    string                 `json:"snapshot_id"`
	NodeID        string                 `json:"node_id"`
	LocalState    map[string]interface{} `json:"local_state"`
	ChannelStates map[string][]string    `json:"channel_states"` // fromNode → recorded messages
	VectorClock   map[string]int         `json:"vector_clock"`
	RecordedAt    time.Time              `json:"recorded_at"`
	Completed     bool                   `json:"completed"`
}

// ChannelRecorder records incoming messages on a channel (from a specific peer)
// between the first marker arrival and the closing marker.
type ChannelRecorder struct {
	FromNode  string   `json:"from_node"`
	Recording bool     `json:"recording"`
	Messages  []string `json:"messages"`
}

// SnapshotManager implements the Chandy-Lamport global snapshot algorithm.
//
// The algorithm works as follows:
//  1. An initiator node records its own local state and sends a MARKER
//     message on all outgoing channels.
//  2. When a node receives a MARKER for the first time:
//     - It records its own local state
//     - It sends MARKERs on all its outgoing channels
//     - It starts recording messages on all incoming channels (except the one
//       the marker came from)
//  3. When a node receives a MARKER on a channel that is already being
//     recorded, it stops recording on that channel. The recorded messages
//     represent the channel state.
//  4. The snapshot is complete when all channels have been closed.
type SnapshotManager struct {
	nodeID    string
	peerNodes []string // IDs of all other nodes
	clock     *VectorClock

	// Active snapshots by snapshot ID
	snapshots map[string]*SnapshotState

	// Channel recorders: snapshotID → fromNode → recorder
	recorders map[string]map[string]*ChannelRecorder

	// Track which snapshots this node has already marked (first marker seen)
	markedSnapshots map[string]bool

	// Callback to get local state (set by the node at init)
	getLocalState func() map[string]interface{}

	mu sync.RWMutex
}

// NewSnapshotManager creates a new snapshot manager.
// getLocalState is a callback that returns the node's current local state
// (e.g., database contents, user count).
func NewSnapshotManager(nodeID string, peerNodes []string, clock *VectorClock, getLocalState func() map[string]interface{}) *SnapshotManager {
	return &SnapshotManager{
		nodeID:          nodeID,
		peerNodes:       peerNodes,
		clock:           clock,
		snapshots:       make(map[string]*SnapshotState),
		recorders:       make(map[string]map[string]*ChannelRecorder),
		markedSnapshots: make(map[string]bool),
		getLocalState:   getLocalState,
	}
}

// InitiateSnapshot starts a new snapshot from this node.
// Returns the list of peer nodes that markers should be sent to.
func (sm *SnapshotManager) InitiateSnapshot(snapshotID string) ([]string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.snapshots[snapshotID]; exists {
		return nil, fmt.Errorf("snapshot %s already exists", snapshotID)
	}

	// Step 1: Record own local state
	localState := sm.getLocalState()
	clockSnapshot := sm.clock.GetClock()

	sm.snapshots[snapshotID] = &SnapshotState{
		SnapshotID:    snapshotID,
		NodeID:        sm.nodeID,
		LocalState:    localState,
		ChannelStates: make(map[string][]string),
		VectorClock:   clockSnapshot,
		RecordedAt:    time.Now(),
		Completed:     false,
	}

	sm.markedSnapshots[snapshotID] = true

	// Step 2: Start recording on all incoming channels
	sm.recorders[snapshotID] = make(map[string]*ChannelRecorder)
	for _, peer := range sm.peerNodes {
		sm.recorders[snapshotID][peer] = &ChannelRecorder{
			FromNode:  peer,
			Recording: true,
			Messages:  make([]string, 0),
		}
	}

	// Step 3: Return peer list — caller sends MARKER to each
	return sm.peerNodes, nil
}

// HandleMarker processes an incoming MARKER message from another node.
// Returns (peersToNotify, isFirstMarker):
//   - If first marker for this snapshot: records state, returns peers to forward marker to
//   - If duplicate marker: closes that channel's recording, returns nil
func (sm *SnapshotManager) HandleMarker(snapshotID string, fromNode string) ([]string, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.markedSnapshots[snapshotID] {
		// First marker for this snapshot — record own state
		localState := sm.getLocalState()
		clockSnapshot := sm.clock.GetClock()

		sm.snapshots[snapshotID] = &SnapshotState{
			SnapshotID:    snapshotID,
			NodeID:        sm.nodeID,
			LocalState:    localState,
			ChannelStates: make(map[string][]string),
			VectorClock:   clockSnapshot,
			RecordedAt:    time.Now(),
			Completed:     false,
		}

		sm.markedSnapshots[snapshotID] = true

		// Start recording on all incoming channels except fromNode
		sm.recorders[snapshotID] = make(map[string]*ChannelRecorder)
		for _, peer := range sm.peerNodes {
			if peer == fromNode {
				// Channel from sender is empty (marker arrived first)
				sm.recorders[snapshotID][peer] = &ChannelRecorder{
					FromNode:  peer,
					Recording: false,
					Messages:  make([]string, 0),
				}
			} else {
				sm.recorders[snapshotID][peer] = &ChannelRecorder{
					FromNode:  peer,
					Recording: true,
					Messages:  make([]string, 0),
				}
			}
		}

		return sm.peerNodes, true
	}

	// Duplicate marker — stop recording on this channel
	if recorders, exists := sm.recorders[snapshotID]; exists {
		if recorder, ok := recorders[fromNode]; ok {
			recorder.Recording = false
		}
	}

	// Check if all channels are closed → snapshot complete
	sm.checkComplete(snapshotID)

	return nil, false
}

// RecordMessage records an incoming message on the appropriate channel
// if a snapshot is actively recording from that sender.
func (sm *SnapshotManager) RecordMessage(fromNode string, message string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, recorders := range sm.recorders {
		if recorder, exists := recorders[fromNode]; exists && recorder.Recording {
			recorder.Messages = append(recorder.Messages, message)
		}
	}
}

// GetSnapshotState returns the snapshot state for a given snapshot ID.
func (sm *SnapshotManager) GetSnapshotState(snapshotID string) (*SnapshotState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.snapshots[snapshotID]
	if !exists {
		return nil, fmt.Errorf("snapshot %s not found", snapshotID)
	}

	// Attach channel states
	if recorders, ok := sm.recorders[snapshotID]; ok {
		for fromNode, recorder := range recorders {
			state.ChannelStates[fromNode] = recorder.Messages
		}
	}

	return state, nil
}

// GetAllSnapshots returns all snapshot IDs and their completion status.
func (sm *SnapshotManager) GetAllSnapshots() map[string]bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]bool)
	for id, state := range sm.snapshots {
		result[id] = state.Completed
	}
	return result
}

// checkComplete checks if all channels for a snapshot are done recording.
func (sm *SnapshotManager) checkComplete(snapshotID string) {
	recorders, exists := sm.recorders[snapshotID]
	if !exists {
		return
	}

	allDone := true
	for _, recorder := range recorders {
		if recorder.Recording {
			allDone = false
			break
		}
	}

	if allDone {
		if state, ok := sm.snapshots[snapshotID]; ok {
			state.Completed = true
			// Finalize channel states
			for fromNode, recorder := range recorders {
				state.ChannelStates[fromNode] = recorder.Messages
			}
		}
	}
}
