package simulator

import (
	"fmt"
	"sync"
	"time"

	"github.com/saintparish4/chaos/nodes"
)

// NetworkSimulator manages the network simulation
type NetworkSimulator struct {
	topology *Topology
	nodes    map[string]*nodes.Node
	mu       sync.RWMutex
	running  bool

	// Message tracking
	messageTracker map[string]*MessageTrack
	trackerMu      sync.RWMutex
}

// MessageTrack tracks a message through the network
type MessageTrack struct {
	Message   *nodes.Message
	StartTime time.Time
	EndTime   time.Time
	Delivered bool
	Path      []string
	HopCount  int
}

// NewNetworkSimulator creates a new network simulator
func NewNetworkSimulator(topo *Topology) *NetworkSimulator {
	sim := &NetworkSimulator{
		topology:       topo,
		nodes:          make(map[string]*nodes.Node),
		messageTracker: make(map[string]*MessageTrack),
	}

	// Create nodes from topology
	for _, nodeCfg := range topo.Config.Nodes {
		node := nodes.NewNode(nodeCfg.ID, nodes.NodeType(nodeCfg.Type), nodeCfg.Capacity)
		sim.nodes[nodeCfg.ID] = node
	}

	// Compute and set routing tables
	sim.UpdateRoutingTables()

	return sim
}

// UpdateRoutingTables recomputes routing tables for all nodes
func (s *NetworkSimulator) UpdateRoutingTables() {
	allTables := ComputeAllRoutingTables(s.topology)

	s.mu.Lock()
	defer s.mu.Unlock()

	for nodeID, table := range allTables {
		if node, exists := s.nodes[nodeID]; exists {
			node.SetRoutingTable(table)
		}
	}
}

// GetNode returns a node by ID
func (s *NetworkSimulator) GetNode(nodeID string) (*nodes.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}
	return node, nil
}

// Start starts the network simulator
func (s *NetworkSimulator) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = true
}

// Stop stops the network simulator
func (s *NetworkSimulator) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
}

// IsRunning returns whether the simulator is running
func (s *NetworkSimulator) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// SendMessage injects a new message into the network
func (s *NetworkSimulator) SendMessage(source, dest string, payload []byte) (string, error) {
	if !s.IsRunning() {
		return "", fmt.Errorf("simulator is not running")
	}

	sourceNode, err := s.GetNode(source)
	if err != nil {
		return "", err
	}

	// Create message
	msgID := fmt.Sprintf("msg-%d-%s-%s", time.Now().UnixNano(), source, dest)
	msg := &nodes.Message{
		ID:          msgID,
		Source:      source,
		Destination: dest,
		Payload:     payload,
		Timestamp:   time.Now(),
		HopCount:    0,
		Path:        []string{source},
	}

	// Track message
	s.trackerMu.Lock()
	s.messageTracker[msgID] = &MessageTrack{
		Message:   msg,
		StartTime: time.Now(),
		Delivered: false,
		Path:      []string{source},
	}
	s.trackerMu.Unlock()

	// Enqueue message
	if err := sourceNode.EnqueueMessage(msg); err != nil {
		return "", err
	}

	return msgID, nil
}

// ForwardMessage forwards a message from one node to the next hop
func (s *NetworkSimulator) ForwardMessage(currentNodeID string) error {
	currentNode, err := s.GetNode(currentNodeID)
	if err != nil {
		return err
	}

	// Dequeue message
	msg, err := currentNode.DequeueMessage()
	if err != nil {
		return err // Queue empty, not an error condition
	}

	// Check if message reached destination
	if msg.Destination == currentNodeID {
		// Message delivered
		s.trackerMu.Lock()
		if track, exists := s.messageTracker[msg.ID]; exists {
			track.Delivered = true
			track.EndTime = time.Now()
			track.HopCount = msg.HopCount
			track.Path = msg.Path
		}
		s.trackerMu.Unlock()
		return nil
	}

	// Get next hop
	nextHop, exists := currentNode.GetNextHop(msg.Destination)
	if !exists {
		return fmt.Errorf("no route to destination %s from %s", msg.Destination, currentNodeID)
	}

	// Get next node
	nextNode, err := s.GetNode(nextHop)
	if err != nil {
		return err
	}

	// Update message
	msg.HopCount++
	msg.Path = append(msg.Path, nextHop)

	// Forward to next node
	if err := nextNode.EnqueueMessage(msg); err != nil {
		return fmt.Errorf("failed to forward message to %s: %w", nextHop, err)
	}

	// Update tracking
	s.trackerMu.Lock()
	if track, exists := s.messageTracker[msg.ID]; exists {
		track.Path = msg.Path
		track.HopCount = msg.HopCount
	}
	s.trackerMu.Unlock()

	currentNode.IncrementMessagesSent()

	return nil
}

// ProcessAllNodes processes messages at all nodes
func (s *NetworkSimulator) ProcessAllNodes() error {
	s.mu.RLock()
	nodeIDs := make([]string, 0, len(s.nodes))
	for nodeID := range s.nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	s.mu.RUnlock()

	for _, nodeID := range nodeIDs {
		// Process all messages in the node's queue
		for {
			err := s.ForwardMessage(nodeID)
			if err != nil {
				break // Queue empty or error
			}
		}
	}

	return nil
}

// GetMessageTrack returns tracking information for a message
func (s *NetworkSimulator) GetMessageTrack(msgID string) (*MessageTrack, error) {
	s.trackerMu.RLock()
	defer s.trackerMu.RUnlock()

	track, exists := s.messageTracker[msgID]
	if !exists {
		return nil, fmt.Errorf("message not found: %s", msgID)
	}

	return track, nil
}

// GetAllNodes returns all nodes in the network
func (s *NetworkSimulator) GetAllNodes() []*nodes.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*nodes.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		result = append(result, node)
	}
	return result
}

// PrintNetworkStatus prints the current status of all nodes
func (s *NetworkSimulator) PrintNetworkStatus() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fmt.Println("\n=== Network Status ===")
	for _, node := range s.nodes {
		sent, received, dropped, queueDepth := node.GetMetrics()
		fmt.Printf("%-10s | State: %-8s | Queue: %2d | Sent: %3d | Recv: %3d | Drop: %3d\n",
			node.ID, node.GetState(), queueDepth, sent, received, dropped)
	}
}

// PrintMessageTracking prints all message tracking information
func (s *NetworkSimulator) PrintMessageTracking() {
	s.trackerMu.RLock()
	defer s.trackerMu.RUnlock()

	fmt.Println("\n=== Message Tracking ===")
	for msgID, track := range s.messageTracker {
		status := "In Transit"
		if track.Delivered {
			latency := track.EndTime.Sub(track.StartTime)
			status = fmt.Sprintf("Delivered in %v", latency)
		}
		fmt.Printf("%-20s | %s -> %s | Hops: %d | Path: %v | Status: %s\n",
			msgID[:20], track.Message.Source, track.Message.Destination,
			track.HopCount, track.Path, status)
	}
}
