package algorithms

import (
	"testing"
)

func TestNewVectorClock(t *testing.T) {
	nodes := []string{"node1", "node2", "node3"}
	vc := NewVectorClock("node1", nodes)

	clock := vc.GetClock()
	for _, n := range nodes {
		if clock[n] != 0 {
			t.Errorf("Expected clock[%s] = 0, got %d", n, clock[n])
		}
	}
	if vc.GetNodeID() != "node1" {
		t.Errorf("Expected node ID 'node1', got '%s'", vc.GetNodeID())
	}
}

func TestVectorClockTick(t *testing.T) {
	nodes := []string{"node1", "node2"}
	vc := NewVectorClock("node1", nodes)

	vc.Tick()
	clock := vc.GetClock()
	if clock["node1"] != 1 {
		t.Errorf("Expected clock[node1] = 1 after tick, got %d", clock["node1"])
	}
	if clock["node2"] != 0 {
		t.Errorf("Expected clock[node2] = 0, got %d", clock["node2"])
	}

	vc.Tick()
	clock = vc.GetClock()
	if clock["node1"] != 2 {
		t.Errorf("Expected clock[node1] = 2 after two ticks, got %d", clock["node1"])
	}
}

func TestVectorClockSend(t *testing.T) {
	nodes := []string{"A", "B", "C"}
	vc := NewVectorClock("A", nodes)

	vc.Tick() // clock[A] = 1
	sent := vc.Send() // clock[A] = 2, returns copy

	if sent["A"] != 2 {
		t.Errorf("Expected sent clock[A] = 2, got %d", sent["A"])
	}

	// Modifying the returned copy should not affect the original
	sent["A"] = 999
	clock := vc.GetClock()
	if clock["A"] != 2 {
		t.Errorf("Send returned a non-copy: modifying it affected the original")
	}
}

func TestVectorClockReceive(t *testing.T) {
	nodes := []string{"A", "B", "C"}
	vcA := NewVectorClock("A", nodes)
	vcB := NewVectorClock("B", nodes)

	// A does some local events
	vcA.Tick() // A:[1,0,0]
	vcA.Tick() // A:[2,0,0]

	// B does a local event
	vcB.Tick() // B:[0,1,0]

	// A sends message to B
	msgClock := vcA.Send() // A:[3,0,0]

	// B receives from A
	vcB.Receive(msgClock) // B: max([0,1,0], [3,0,0]) + tick = [3,2,0]

	clockB := vcB.GetClock()
	if clockB["A"] != 3 {
		t.Errorf("Expected B's clock[A] = 3 after receive, got %d", clockB["A"])
	}
	if clockB["B"] != 2 {
		t.Errorf("Expected B's clock[B] = 2 after receive, got %d", clockB["B"])
	}
}

func TestVectorClockCompareBefore(t *testing.T) {
	nodes := []string{"A", "B"}
	vc := NewVectorClock("A", nodes)

	vc.Tick() // A:[1,0]

	other := map[string]int{"A": 2, "B": 1}

	result := vc.Compare(other)
	if result != BEFORE {
		t.Errorf("Expected BEFORE, got %s", result)
	}
}

func TestVectorClockCompareAfter(t *testing.T) {
	nodes := []string{"A", "B"}
	vc := NewVectorClock("A", nodes)

	vc.Tick() // A:[1,0]
	vc.Tick() // A:[2,0]

	other := map[string]int{"A": 1, "B": 0}

	result := vc.Compare(other)
	if result != AFTER {
		t.Errorf("Expected AFTER, got %s", result)
	}
}

func TestVectorClockCompareConcurrent(t *testing.T) {
	nodes := []string{"A", "B"}
	vc := NewVectorClock("A", nodes)

	vc.Tick() // A:[1,0]

	other := map[string]int{"A": 0, "B": 1}

	result := vc.Compare(other)
	if result != CONCURRENT {
		t.Errorf("Expected CONCURRENT, got %s", result)
	}
}

func TestVectorClockCompareEqual(t *testing.T) {
	nodes := []string{"A", "B"}
	vc := NewVectorClock("A", nodes)

	other := map[string]int{"A": 0, "B": 0}
	result := vc.Compare(other)
	if result != EQUAL {
		t.Errorf("Expected EQUAL, got %s", result)
	}
}

func TestEventLog(t *testing.T) {
	nodes := []string{"node1", "node2"}
	vc := NewVectorClock("node1", nodes)
	el := NewEventLog("node1", vc)

	el.LogEvent("INSERT", "Created user Alice")
	el.LogEvent("READ", "Read user Bob")

	events := el.GetEvents()
	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	if events[0].EventType != "INSERT" {
		t.Errorf("Expected first event type INSERT, got %s", events[0].EventType)
	}
	if events[1].EventType != "READ" {
		t.Errorf("Expected second event type READ, got %s", events[1].EventType)
	}

	// Vector clock should have been ticked twice
	if events[1].Timestamp["node1"] != 2 {
		t.Errorf("Expected timestamp[node1] = 2 for second event, got %d", events[1].Timestamp["node1"])
	}
}

func TestEventLogSince(t *testing.T) {
	nodes := []string{"n1"}
	vc := NewVectorClock("n1", nodes)
	el := NewEventLog("n1", vc)

	el.LogEvent("A", "first")
	el.LogEvent("B", "second")
	el.LogEvent("C", "third")

	events := el.GetEventsSince(1)
	if len(events) != 2 {
		t.Errorf("Expected 2 events since index 1, got %d", len(events))
	}

	events = el.GetEventsSince(10)
	if events != nil {
		t.Errorf("Expected nil for out-of-range index, got %v", events)
	}
}
