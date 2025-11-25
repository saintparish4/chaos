package nodes

import (
	"fmt"
	"sync"
	"time"
)

// NodeState represents the current state of a node
type NodeState string

const (
	StateActive   NodeState = "ACTIVE"
	StateDegraded NodeState = "DEGRADED"
	StateFailed   NodeState = "FAILED"
)

// NodeType represents the type of network node
type NodeType string

const (
	NodeTypeRouter NodeType = "router"
	NodeTypeSwitch NodeType = "switch"
	NodeTypeTower  NodeType = "tower"
)

// Message represents a network message
type Message struct {
	ID          string
	Source      string
	Destination string
	Payload     []byte
	Timestamp   time.Time
	HopCount    int
	Path        []string // Track the path the message has taken
}

// Node represents a network node
type Node struct {
	ID       string
	Type     NodeType
	State    NodeState
	Capacity int // messages per second

	// Message queue
	queue     chan *Message
	queueSize int
	mu        sync.RWMutex

	// Routing table
	routingTable map[string]string // destination -> next hop

	// Metrics
	messagesSent     int64
	messagesReceived int64
	messagesDropped  int64
}

// NewNode creates a new network node
func NewNode(id string, nodeType NodeType, capacity int) *Node {
	return &Node{
		ID:           id,
		Type:         nodeType,
		State:        StateActive,
		Capacity:     capacity,
		queue:        make(chan *Message, 100), // buffered queue
		queueSize:    100,
		routingTable: make(map[string]string),
	}
}

// SetState updates the node state
func (n *Node) SetState(state NodeState) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.State = state
}

// GetState returns the current node state
func (n *Node) GetState() NodeState {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.State
}

// EnqueueMessage adds a message to the node's queue
func (n *Node) EnqueueMessage(msg *Message) error {
	n.mu.RLock()
	state := n.State
	n.mu.RUnlock()

	if state == StateFailed {
		n.mu.Lock()
		n.messagesDropped++
		n.mu.Unlock()
		return fmt.Errorf("node %s is in FAILED state", n.ID)
	}

	select {
	case n.queue <- msg:
		n.mu.Lock()
		n.messagesReceived++
		n.mu.Unlock()
		return nil
	default:
		n.mu.Lock()
		n.messagesDropped++
		n.mu.Unlock()
		return fmt.Errorf("node %s queue is full", n.ID)
	}
}

// DequeueMessage retrieves the next message from the queue
func (n *Node) DequeueMessage() (*Message, error) {
	select {
	case msg := <-n.queue:
		return msg, nil
	default:
		return nil, fmt.Errorf("queue is empty")
	}
}

// QueueDepth returns the current number of messages in the queue
func (n *Node) QueueDepth() int {
	return len(n.queue)
}

// SetRoutingTable updates the entire routing table
func (n *Node) SetRoutingTable(table map[string]string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.routingTable = table
}

// GetNextHop returns the next hop for a destination
func (n *Node) GetNextHop(dest string) (string, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	nextHop, exists := n.routingTable[dest]
	return nextHop, exists
}

// UpdateRoutingEntry updates a single routing table entry
func (n *Node) UpdateRoutingEntry(dest, nextHop string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.routingTable[dest] = nextHop
}

// IncrementMessagesSent increments the sent message counter
func (n *Node) IncrementMessagesSent() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messagesSent++
}

// GetMetrics returns current node metrics
func (n *Node) GetMetrics() (sent, received, dropped int64, queueDepth int) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.messagesSent, n.messagesReceived, n.messagesDropped, len(n.queue)
}

// String returns a string representation of the node
func (n *Node) String() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return fmt.Sprintf("Node{ID: %s, Type: %s, State: %s, QueueDepth: %d/%d}", n.ID, n.Type, n.State, len(n.queue), n.queueSize)
}
