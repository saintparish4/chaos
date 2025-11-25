package nodes

import (
	"testing"
	"time"
)

func TestNewNode(t *testing.T) {
	node := NewNode("test-node", NodeTypeRouter, 1000)

	if node.ID != "test-node" {
		t.Errorf("Expected ID 'test-node', got '%s'", node.ID)
	}

	if node.Type != NodeTypeRouter {
		t.Errorf("Expected type Router, got %s", node.Type)
	}

	if node.State != StateActive {
		t.Errorf("Expected initial state ACTIVE, got %s", node.State)
	}

	if node.Capacity != 1000 {
		t.Errorf("Expected capacity 1000, got %d", node.Capacity)
	}
}

func TestNodeStateTransitions(t *testing.T) {
	node := NewNode("test-node", NodeTypeRouter, 1000)

	// Test state change to DEGRADED
	node.SetState(StateDegraded)
	if node.GetState() != StateDegraded {
		t.Errorf("Expected state DEGRADED, got %s", node.GetState())
	}

	// Test state change to FAILED
	node.SetState(StateFailed)
	if node.GetState() != StateFailed {
		t.Errorf("Expected state FAILED, got %s", node.GetState())
	}

	// Test state change back to ACTIVE
	node.SetState(StateActive)
	if node.GetState() != StateActive {
		t.Errorf("Expected state ACTIVE, got %s", node.GetState())
	}
}

func TestMessageQueue(t *testing.T) {
	node := NewNode("test-node", NodeTypeRouter, 1000)

	msg := &Message{
		ID:          "msg-1",
		Source:      "source",
		Destination: "dest",
		Payload:     []byte("test"),
		Timestamp:   time.Now(),
	}

	// Test enqueue
	err := node.EnqueueMessage(msg)
	if err != nil {
		t.Errorf("Failed to enqueue message: %v", err)
	}

	if node.QueueDepth() != 1 {
		t.Errorf("Expected queue depth 1, got %d", node.QueueDepth())
	}

	// Test dequeue
	dequeuedMsg, err := node.DequeueMessage()
	if err != nil {
		t.Errorf("Failed to dequeue message: %v", err)
	}

	if dequeuedMsg.ID != msg.ID {
		t.Errorf("Expected message ID %s, got %s", msg.ID, dequeuedMsg.ID)
	}

	if node.QueueDepth() != 0 {
		t.Errorf("Expected queue depth 0, got %d", node.QueueDepth())
	}
}

func TestFailedNodeRejectsMessages(t *testing.T) {
	node := NewNode("test-node", NodeTypeRouter, 1000)
	node.SetState(StateFailed)

	msg := &Message{
		ID:          "msg-1",
		Source:      "source",
		Destination: "dest",
		Payload:     []byte("test"),
		Timestamp:   time.Now(),
	}

	err := node.EnqueueMessage(msg)
	if err == nil {
		t.Error("Expected error when enqueueing to failed node, got nil")
	}

	sent, received, dropped, _ := node.GetMetrics()
	if dropped != 1 {
		t.Errorf("Expected 1 dropped message, got %d", dropped)
	}

	if sent != 0 || received != 0 {
		t.Errorf("Expected 0 sent/received, got sent=%d, received=%d", sent, received)
	}
}

func TestQueueFull(t *testing.T) {
	node := NewNode("test-node", NodeTypeRouter, 1000)

	// Fill the queue (queue size is 100)
	for i := 0; i < 100; i++ {
		msg := &Message{
			ID:          string(rune(i)),
			Source:      "source",
			Destination: "dest",
			Payload:     []byte("test"),
			Timestamp:   time.Now(),
		}
		err := node.EnqueueMessage(msg)
		if err != nil {
			t.Fatalf("Failed to enqueue message %d: %v", i, err)
		}
	}

	// Try to add one more - should fail
	msg := &Message{
		ID:          "overflow",
		Source:      "source",
		Destination: "dest",
		Payload:     []byte("test"),
		Timestamp:   time.Now(),
	}

	err := node.EnqueueMessage(msg)
	if err == nil {
		t.Error("Expected error when queue is full, got nil")
	}

	sent, received, dropped, depth := node.GetMetrics()
	if dropped != 1 {
		t.Errorf("Expected 1 dropped message, got %d", dropped)
	}
	if received != 100 {
		t.Errorf("Expected 100 received messages, got %d", received)
	}
	if depth != 100 {
		t.Errorf("Expected queue depth 100, got %d", depth)
	}
	if sent != 0 {
		t.Errorf("Expected 0 sent messages, got %d", sent)
	}
}

func TestRoutingTable(t *testing.T) {
	node := NewNode("test-node", NodeTypeRouter, 1000)

	// Set routing table
	table := map[string]string{
		"dest-1": "next-hop-1",
		"dest-2": "next-hop-2",
		"dest-3": "next-hop-3",
	}
	node.SetRoutingTable(table)

	// Test getting next hop
	nextHop, exists := node.GetNextHop("dest-1")
	if !exists {
		t.Error("Expected route to dest-1 to exist")
	}
	if nextHop != "next-hop-1" {
		t.Errorf("Expected next hop 'next-hop-1', got '%s'", nextHop)
	}

	// Test non-existent route
	_, exists = node.GetNextHop("dest-99")
	if exists {
		t.Error("Expected route to dest-99 to not exist")
	}

	// Test updating single entry
	node.UpdateRoutingEntry("dest-4", "next-hop-4")
	nextHop, exists = node.GetNextHop("dest-4")
	if !exists {
		t.Error("Expected route to dest-4 to exist after update")
	}
	if nextHop != "next-hop-4" {
		t.Errorf("Expected next hop 'next-hop-4', got '%s'", nextHop)
	}
}

func TestMessageMetrics(t *testing.T) {
	node := NewNode("test-node", NodeTypeRouter, 1000)

	// Enqueue some messages
	for i := 0; i < 5; i++ {
		msg := &Message{
			ID:          string(rune(i)),
			Source:      "source",
			Destination: "dest",
			Payload:     []byte("test"),
			Timestamp:   time.Now(),
		}
		node.EnqueueMessage(msg)
	}

	// Increment sent counter
	for i := 0; i < 3; i++ {
		node.IncrementMessagesSent()
	}

	sent, received, dropped, depth := node.GetMetrics()

	if sent != 3 {
		t.Errorf("Expected 3 sent messages, got %d", sent)
	}
	if received != 5 {
		t.Errorf("Expected 5 received messages, got %d", received)
	}
	if dropped != 0 {
		t.Errorf("Expected 0 dropped messages, got %d", dropped)
	}
	if depth != 5 {
		t.Errorf("Expected queue depth 5, got %d", depth)
	}
}

func TestNodeString(t *testing.T) {
	node := NewNode("router-1", NodeTypeRouter, 1000)
	str := node.String()

	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	// String should contain node ID
	if len(str) < 5 {
		t.Errorf("Expected meaningful string representation, got: %s", str)
	}
}
