package algorithms

import (
	"fmt"
	"sync"
	"time"
)

// ClockComparison represents the result of comparing two vector clocks
type ClockComparison int

const (
	BEFORE     ClockComparison = iota // A happened before B
	AFTER                             // A happened after B
	CONCURRENT                        // A and B are concurrent
	EQUAL                             // A and B are equal
)

func (c ClockComparison) String() string {
	switch c {
	case BEFORE:
		return "BEFORE"
	case AFTER:
		return "AFTER"
	case CONCURRENT:
		return "CONCURRENT"
	case EQUAL:
		return "EQUAL"
	default:
		return "UNKNOWN"
	}
}

// VectorClock implements a vector clock for distributed event ordering.
// Each node maintains a vector of counters, one per node in the system.
// The clock captures causal relationships between events:
//   - If event A's clock ≤ event B's clock, then A happened-before B
//   - If neither A ≤ B nor B ≤ A, the events are concurrent
type VectorClock struct {
	nodeID string
	clock  map[string]int
	mu     sync.RWMutex
}

// NewVectorClock creates a new vector clock for the given node.
// allNodeIDs specifies all nodes in the system to initialize counters.
func NewVectorClock(nodeID string, allNodeIDs []string) *VectorClock {
	vc := &VectorClock{
		nodeID: nodeID,
		clock:  make(map[string]int),
	}
	for _, id := range allNodeIDs {
		vc.clock[id] = 0
	}
	return vc
}

// Tick increments this node's own counter (local event).
// Called before logging any local event.
func (vc *VectorClock) Tick() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.clock[vc.nodeID]++
}

// Send increments own counter and returns a copy of the clock
// to be attached to outgoing messages.
func (vc *VectorClock) Send() map[string]int {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.clock[vc.nodeID]++
	return vc.copyClockUnsafe()
}

// Receive merges an incoming clock with the local clock by taking
// element-wise maximum, then increments own counter.
// This captures the causal dependency on the sending event.
func (vc *VectorClock) Receive(remoteClock map[string]int) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	for nodeID, remoteVal := range remoteClock {
		if localVal, exists := vc.clock[nodeID]; !exists || remoteVal > localVal {
			vc.clock[nodeID] = remoteVal
		}
	}
	vc.clock[vc.nodeID]++
}

// GetClock returns a copy of the current clock state.
func (vc *VectorClock) GetClock() map[string]int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.copyClockUnsafe()
}

// GetNodeID returns this clock's node ID.
func (vc *VectorClock) GetNodeID() string {
	return vc.nodeID
}

// Compare compares this clock against another.
//   - BEFORE:     this happened-before other (all entries ≤, at least one <)
//   - AFTER:      other happened-before this
//   - CONCURRENT: neither happened-before the other
//   - EQUAL:      identical clocks
func (vc *VectorClock) Compare(other map[string]int) ClockComparison {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	lessOrEqual := true
	greaterOrEqual := true
	equal := true

	// Collect all keys from both clocks
	allKeys := make(map[string]bool)
	for k := range vc.clock {
		allKeys[k] = true
	}
	for k := range other {
		allKeys[k] = true
	}

	for k := range allKeys {
		localVal := vc.clock[k]
		otherVal := other[k]

		if localVal < otherVal {
			greaterOrEqual = false
			equal = false
		}
		if localVal > otherVal {
			lessOrEqual = false
			equal = false
		}
	}

	if equal {
		return EQUAL
	}
	if lessOrEqual {
		return BEFORE
	}
	if greaterOrEqual {
		return AFTER
	}
	return CONCURRENT
}

func (vc *VectorClock) copyClockUnsafe() map[string]int {
	copy := make(map[string]int, len(vc.clock))
	for k, v := range vc.clock {
		copy[k] = v
	}
	return copy
}

// =============================================
// Event Log — records events with vector clock timestamps
// =============================================

// Event represents a logged event with its vector clock timestamp.
type Event struct {
	Timestamp   map[string]int `json:"timestamp"`
	NodeID      string         `json:"node_id"`
	EventType   string         `json:"event_type"`
	Description string         `json:"description"`
	WallClock   time.Time      `json:"wall_clock"`
}

// EventLog maintains an ordered log of events with vector clock timestamps.
type EventLog struct {
	nodeID string
	clock  *VectorClock
	events []Event
	mu     sync.RWMutex
}

// NewEventLog creates an event log associated with a vector clock.
func NewEventLog(nodeID string, clock *VectorClock) *EventLog {
	return &EventLog{
		nodeID: nodeID,
		clock:  clock,
		events: make([]Event, 0),
	}
}

// LogEvent records an event. Ticks the vector clock and captures
// the timestamp at the moment of the event.
func (el *EventLog) LogEvent(eventType, description string) {
	el.clock.Tick()

	event := Event{
		Timestamp:   el.clock.GetClock(),
		NodeID:      el.nodeID,
		EventType:   eventType,
		Description: description,
		WallClock:   time.Now(),
	}

	el.mu.Lock()
	el.events = append(el.events, event)
	el.mu.Unlock()
}

// GetEvents returns all logged events.
func (el *EventLog) GetEvents() []Event {
	el.mu.RLock()
	defer el.mu.RUnlock()

	result := make([]Event, len(el.events))
	copy(result, el.events)
	return result
}

// GetEventsSince returns events after a given index (for pagination).
func (el *EventLog) GetEventsSince(index int) []Event {
	el.mu.RLock()
	defer el.mu.RUnlock()

	if index >= len(el.events) {
		return nil
	}
	result := make([]Event, len(el.events)-index)
	copy(result, el.events[index:])
	return result
}

// String returns a human-readable representation of the vector clock.
func (vc *VectorClock) String() string {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return fmt.Sprintf("VectorClock[%s]%v", vc.nodeID, vc.clock)
}
