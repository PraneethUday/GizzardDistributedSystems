package algorithms

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ElectionMessageType represents the type of an election message in the Bully algorithm.
type ElectionMessageType int

const (
	ELECTION ElectionMessageType = iota // "I'm starting an election"
	OK                                  // "I'm alive and have higher ID, back off"
	VICTORY                             // "I am the new leader"
)

func (e ElectionMessageType) String() string {
	switch e {
	case ELECTION:
		return "ELECTION"
	case OK:
		return "OK"
	case VICTORY:
		return "VICTORY"
	default:
		return "UNKNOWN"
	}
}

// ElectionMessage represents a message exchanged during a Bully election.
type ElectionMessage struct {
	Type     ElectionMessageType `json:"type"`
	FromNode int                 `json:"from_node"`
	Term     int                 `json:"term"` // election term counter
}

// ElectionState holds the current election state of a node.
type ElectionState struct {
	NodeID          int       `json:"node_id"`
	CurrentLeader   int       `json:"current_leader"`
	IsLeader        bool      `json:"is_leader"`
	ElectionTerm    int       `json:"election_term"`
	ElectionActive  bool      `json:"election_active"`
	LastElectionAt  time.Time `json:"last_election_at"`
	VictoryRecvAt   time.Time `json:"victory_received_at,omitempty"`
}

// SendElectionFunc is a callback to send an election message to another node.
// Returns true if the node responded (is alive), false if unreachable.
type SendElectionFunc func(toNodeID int, msg ElectionMessage) bool

// BullyElection implements the Bully leader election algorithm.
//
// The algorithm works as follows:
//  1. When a node detects the leader is down (or starts an election):
//     - It sends ELECTION messages to all nodes with higher IDs
//     - If no higher-ID node responds with OK within a timeout, it declares
//       itself leader and sends VICTORY to all nodes
//     - If a higher-ID node responds, that node takes over the election
//  2. When a node receives an ELECTION message from a lower-ID node:
//     - It responds with OK
//     - It starts its own election (since it has a higher ID)
//  3. When a node receives a VICTORY message:
//     - It accepts the sender as the new leader
//
// The node with the highest ID always wins (hence "Bully").
type BullyElection struct {
	nodeID        int
	allNodeIDs    []int
	currentLeader int
	electionTerm  int
	isElecting    bool
	sendFunc      SendElectionFunc
	timeout       time.Duration
	mu            sync.RWMutex
	lastElection  time.Time
	lastVictory   time.Time
}

// NewBullyElection creates a new Bully election manager.
// sendFunc is a callback used to send messages to other nodes.
func NewBullyElection(nodeID int, allNodeIDs []int, sendFunc SendElectionFunc) *BullyElection {
	// Initially, the highest-ID node is the leader
	maxID := 0
	for _, id := range allNodeIDs {
		if id > maxID {
			maxID = id
		}
	}

	return &BullyElection{
		nodeID:        nodeID,
		allNodeIDs:    allNodeIDs,
		currentLeader: maxID,
		electionTerm:  0,
		isElecting:    false,
		sendFunc:      sendFunc,
		timeout:       3 * time.Second,
	}
}

// StartElection initiates a Bully election from this node.
// Returns the ID of the elected leader.
func (be *BullyElection) StartElection() int {
	be.mu.Lock()
	be.isElecting = true
	be.electionTerm++
	term := be.electionTerm
	be.lastElection = time.Now()
	be.mu.Unlock()

	log.Printf("[ELECTION] Node %d starting election (term %d)", be.nodeID, term)

	// Find all nodes with higher IDs
	higherNodes := be.getHigherNodes()

	if len(higherNodes) == 0 {
		// No higher nodes — I am the highest, declare victory
		log.Printf("[ELECTION] Node %d is the highest — declaring VICTORY", be.nodeID)
		be.declareVictory(term)
		return be.nodeID
	}

	// Send ELECTION to all higher-ID nodes
	gotResponse := false
	msg := ElectionMessage{
		Type:     ELECTION,
		FromNode: be.nodeID,
		Term:     term,
	}

	responseCh := make(chan bool, len(higherNodes))

	for _, higherID := range higherNodes {
		go func(targetID int) {
			log.Printf("[ELECTION] Node %d sending ELECTION to Node %d", be.nodeID, targetID)
			alive := be.sendFunc(targetID, msg)
			responseCh <- alive
		}(higherID)
	}

	// Wait for responses with timeout
	timer := time.NewTimer(be.timeout)
	defer timer.Stop()

	responsesReceived := 0
	for responsesReceived < len(higherNodes) {
		select {
		case alive := <-responseCh:
			responsesReceived++
			if alive {
				gotResponse = true
			}
		case <-timer.C:
			goto done
		}
	}

done:
	if !gotResponse {
		// No higher node responded — I am the leader
		log.Printf("[ELECTION] No higher node responded — Node %d declares VICTORY", be.nodeID)
		be.declareVictory(term)
		return be.nodeID
	}

	// A higher node responded — it will take over, wait for VICTORY
	log.Printf("[ELECTION] Higher node responded — Node %d waiting for VICTORY", be.nodeID)
	be.mu.Lock()
	be.isElecting = false
	be.mu.Unlock()

	return be.GetLeader()
}

// HandleElectionMessage processes an incoming ELECTION message from a lower-ID node.
// Returns true if this node should start its own election (it has a higher ID).
func (be *BullyElection) HandleElectionMessage(fromNode int, term int) bool {
	be.mu.Lock()
	defer be.mu.Unlock()

	log.Printf("[ELECTION] Node %d received ELECTION from Node %d (term %d)", be.nodeID, fromNode, term)

	if be.nodeID > fromNode {
		// I have a higher ID — send OK (done implicitly by returning true)
		// and start my own election
		log.Printf("[ELECTION] Node %d has higher ID than %d — will respond OK and start own election", be.nodeID, fromNode)
		return true
	}

	return false
}

// HandleVictoryMessage processes an incoming VICTORY message.
func (be *BullyElection) HandleVictoryMessage(leaderID int, term int) {
	be.mu.Lock()
	defer be.mu.Unlock()

	log.Printf("[ELECTION] Node %d received VICTORY — new leader is Node %d (term %d)", be.nodeID, leaderID, term)

	be.currentLeader = leaderID
	be.isElecting = false
	be.lastVictory = time.Now()

	if term > be.electionTerm {
		be.electionTerm = term
	}
}

// declareVictory announces this node as the leader to all other nodes.
func (be *BullyElection) declareVictory(term int) {
	be.mu.Lock()
	be.currentLeader = be.nodeID
	be.isElecting = false
	be.mu.Unlock()

	msg := ElectionMessage{
		Type:     VICTORY,
		FromNode: be.nodeID,
		Term:     term,
	}

	for _, nodeID := range be.allNodeIDs {
		if nodeID != be.nodeID {
			go func(targetID int) {
				log.Printf("[ELECTION] Node %d sending VICTORY to Node %d", be.nodeID, targetID)
				be.sendFunc(targetID, msg)
			}(nodeID)
		}
	}
}

// getHigherNodes returns all node IDs higher than this node.
func (be *BullyElection) getHigherNodes() []int {
	var higher []int
	for _, id := range be.allNodeIDs {
		if id > be.nodeID {
			higher = append(higher, id)
		}
	}
	return higher
}

// GetLeader returns the current leader's node ID.
func (be *BullyElection) GetLeader() int {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.currentLeader
}

// IsLeader returns true if this node is the current leader.
func (be *BullyElection) IsLeader() bool {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.currentLeader == be.nodeID
}

// GetState returns the current election state.
func (be *BullyElection) GetState() ElectionState {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return ElectionState{
		NodeID:         be.nodeID,
		CurrentLeader:  be.currentLeader,
		IsLeader:       be.currentLeader == be.nodeID,
		ElectionTerm:   be.electionTerm,
		ElectionActive: be.isElecting,
		LastElectionAt: be.lastElection,
		VictoryRecvAt:  be.lastVictory,
	}
}

// SetTimeout sets the election response timeout.
func (be *BullyElection) SetTimeout(d time.Duration) {
	be.mu.Lock()
	defer be.mu.Unlock()
	be.timeout = d
}

// String returns a human-readable status.
func (be *BullyElection) String() string {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return fmt.Sprintf("BullyElection[Node %d, Leader: %d, Term: %d, Electing: %v]",
		be.nodeID, be.currentLeader, be.electionTerm, be.isElecting)
}
