package algorithms

import (
	"testing"
	"time"
)

func TestBullyElectionHighestNodeWins(t *testing.T) {
	// Node 4 is the highest — should win immediately
	sendFunc := func(toNodeID int, msg ElectionMessage) bool {
		return false // no higher node responds
	}

	election := NewBullyElection(4, []int{1, 2, 3, 4}, sendFunc)
	election.SetTimeout(500 * time.Millisecond)

	leader := election.StartElection()
	if leader != 4 {
		t.Errorf("Expected node 4 to win, got %d", leader)
	}
	if !election.IsLeader() {
		t.Error("Expected node 4 to be leader")
	}
}

func TestBullyElectionNoHigherNodesRespond(t *testing.T) {
	// Node 2 starts election, nodes 3 and 4 don't respond
	sendFunc := func(toNodeID int, msg ElectionMessage) bool {
		return false // simulate no response (nodes down)
	}

	election := NewBullyElection(2, []int{1, 2, 3, 4}, sendFunc)
	election.SetTimeout(500 * time.Millisecond)

	leader := election.StartElection()
	if leader != 2 {
		t.Errorf("Expected node 2 to win when higher nodes are down, got %d", leader)
	}
}

func TestBullyElectionHigherNodeResponds(t *testing.T) {
	// Node 1 starts election, node 3 responds (alive)
	sendFunc := func(toNodeID int, msg ElectionMessage) bool {
		if msg.Type == ELECTION {
			return true // higher node responds OK
		}
		return true
	}

	election := NewBullyElection(1, []int{1, 2, 3, 4}, sendFunc)
	election.SetTimeout(500 * time.Millisecond)

	election.StartElection()
	// Node 1 should NOT be leader since higher nodes responded
	if election.IsLeader() {
		t.Error("Node 1 should NOT be leader when higher nodes respond")
	}
}

func TestHandleElectionMessage(t *testing.T) {
	sendFunc := func(toNodeID int, msg ElectionMessage) bool { return true }
	election := NewBullyElection(3, []int{1, 2, 3, 4}, sendFunc)

	// Node 3 receives ELECTION from Node 1 (lower ID)
	shouldStartElection := election.HandleElectionMessage(1, 1)
	if !shouldStartElection {
		t.Error("Node 3 (higher ID) should start its own election when receiving from Node 1")
	}

	// Node 3 receives ELECTION from Node 4 (higher ID)
	shouldStartElection = election.HandleElectionMessage(4, 1)
	if shouldStartElection {
		t.Error("Node 3 should NOT start election when receiving from higher Node 4")
	}
}

func TestHandleVictoryMessage(t *testing.T) {
	sendFunc := func(toNodeID int, msg ElectionMessage) bool { return true }
	election := NewBullyElection(2, []int{1, 2, 3, 4}, sendFunc)

	// Node 2 receives victory from Node 4
	election.HandleVictoryMessage(4, 1)

	if election.GetLeader() != 4 {
		t.Errorf("Expected leader to be 4 after victory message, got %d", election.GetLeader())
	}
	if election.IsLeader() {
		t.Error("Node 2 should not think it's the leader after node 4's victory")
	}
}

func TestGetElectionState(t *testing.T) {
	sendFunc := func(toNodeID int, msg ElectionMessage) bool { return false }
	election := NewBullyElection(3, []int{1, 2, 3, 4}, sendFunc)

	state := election.GetState()
	if state.NodeID != 3 {
		t.Errorf("Expected node ID 3, got %d", state.NodeID)
	}
	// Initially, highest node (4) is the leader
	if state.CurrentLeader != 4 {
		t.Errorf("Expected initial leader to be 4, got %d", state.CurrentLeader)
	}
}

func TestBullyElectionString(t *testing.T) {
	sendFunc := func(toNodeID int, msg ElectionMessage) bool { return false }
	election := NewBullyElection(2, []int{1, 2, 3, 4}, sendFunc)

	s := election.String()
	if s == "" {
		t.Error("Expected non-empty string representation")
	}
}
